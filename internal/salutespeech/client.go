package salutespeech

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dmnAlex/sberscribe/internal/auth"
	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/pkg/api/recognitionv1"
	"github.com/dmnAlex/sberscribe/pkg/api/storagev1"
	"github.com/dmnAlex/sberscribe/pkg/api/taskv1"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	saluteAddress = "smartspeech.sber.ru:443"
	chunkSize     = 64 * 1024
)

type Client struct {
	tokenMgr *auth.TokenManager
	conn     *grpc.ClientConn

	storageClient storagev1.SmartSpeechClient
	recognClient  recognitionv1.SmartSpeechClient
	taskClient    taskv1.SmartSpeechClient
}

func NewSaluteClient(tokenMgr *auth.TokenManager, tlsConfig *tls.Config) (*Client, error) {
	conn, err := grpc.NewClient(saluteAddress, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, errors.Wrap(err, "new grpc client")
	}
	return &Client{
		tokenMgr:      tokenMgr,
		conn:          conn,
		storageClient: storagev1.NewSmartSpeechClient(conn),
		recognClient:  recognitionv1.NewSmartSpeechClient(conn),
		taskClient:    taskv1.NewSmartSpeechClient(conn),
	}, nil
}

func (c *Client) Close() { c.conn.Close() }

func (c *Client) Upload(ctx context.Context, data io.Reader) (string, error) {
	ctx, err := c.prepareContext(ctx)
	if err != nil {
		return "", errors.Wrap(err, "prepare context")
	}

	return c.upload(ctx, data)
}

func (c *Client) Recognize(ctx context.Context, uploadFileID, mimeType string) (string, error) {
	ctx, err := c.prepareContext(ctx)
	if err != nil {
		return "", errors.Wrap(err, "prepare context")
	}

	return c.asyncRecognize(ctx, uploadFileID, mimeType)
}

func (c *Client) PollTask(ctx context.Context, taskID string) (string, error) {
	ctx, err := c.prepareContext(ctx)
	if err != nil {
		return "", errors.Wrap(err, "prepare context")
	}

	return c.pollTask(ctx, taskID)
}

func (c *Client) Download(ctx context.Context, downloadFileID string) (string, []byte, error) {
	ctx, err := c.prepareContext(ctx)
	if err != nil {
		return "", nil, errors.Wrap(err, "prepare context")
	}

	raw, err := c.download(ctx, downloadFileID)
	if err != nil {
		return "", nil, errors.Wrap(err, "download")
	}

	text, err := extractText(raw)
	return text, raw, errors.Wrap(err, "extract text")
}

func (c *Client) upload(ctx context.Context, audio io.Reader) (string, error) {
	stream, err := c.storageClient.Upload(ctx)
	if err != nil {
		return "", errors.Wrap(err, "upload to storage")
	}

	buf := make([]byte, chunkSize)
	for {
		n, err := audio.Read(buf)
		if n > 0 {
			req := storagev1.UploadRequest_builder{
				FileChunk: buf[:n],
			}.Build()

			if sendErr := stream.Send(req); sendErr != nil {
				return "", errors.Wrap(sendErr, "send stream")
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return "", errors.Wrap(err, "read audio")
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		return "", errors.Wrap(err, "close and receive")
	}

	return res.GetRequestFileId(), nil
}

func (c *Client) asyncRecognize(ctx context.Context, requestFileID, mimeType string) (string, error) {
	var encoding recognitionv1.RecognitionOptions_AudioEncoding
	switch mimeType {
	case "audio/ogg":
		encoding = recognitionv1.RecognitionOptions_OPUS
	case "audio/mpeg":
		encoding = recognitionv1.RecognitionOptions_MP3
	case "audio/flac":
		encoding = recognitionv1.RecognitionOptions_FLAC
	default:
		return "", errors.Errorf("unsupported mime type: %s", mimeType)
	}

	opts := recognitionv1.RecognitionOptions_builder{
		AudioEncoding: encoding,
		Language:      "ru-RU",
		Model:         "general",
	}.Build()

	req := recognitionv1.AsyncRecognizeRequest_builder{
		Options:       opts,
		RequestFileId: requestFileID,
	}.Build()

	task, err := c.recognClient.AsyncRecognize(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "async recognize")
	}

	return task.GetId(), nil
}

func (c *Client) pollTask(ctx context.Context, taskID string) (string, error) {
	req := taskv1.GetTaskRequest_builder{
		TaskId: taskID,
	}.Build()

	res, err := c.taskClient.GetTask(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "get task")
	}

	switch res.GetStatus() {
	case taskv1.Task_DONE:
		if res.GetResponseFileId() == "" {
			return "", errors.Errorf("empty response_file_id")
		}
	case taskv1.Task_ERROR:
		msg := fmt.Sprintf("recognition error: %s", res.GetError())
		return "", model.NewErrorWithStatus(msg, model.ErrorFailed)
	case taskv1.Task_CANCELED:
		msg := "task canceled"
		return "", model.NewErrorWithStatus(msg, model.ErrorCanceled)
	}

	return res.GetResponseFileId(), nil
}

func (c *Client) download(ctx context.Context, responseFileID string) ([]byte, error) {
	req := storagev1.DownloadRequest_builder{
		ResponseFileId: responseFileID,
	}.Build()

	stream, err := c.storageClient.Download(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "download")
	}

	var buf bytes.Buffer
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "receive stream")
		}
		buf.Write(chunk.GetFileChunk())
	}
	return buf.Bytes(), nil
}

func extractText(raw []byte) (string, error) {
	var results []struct {
		Results []struct {
			Text string `json:"text"`
		} `json:"results"`
	}

	if err := json.Unmarshal(raw, &results); err != nil {
		return "", errors.Wrap(err, "unmarshal json")
	}

	var sb strings.Builder
	for _, outer := range results {
		for _, inner := range outer.Results {
			sb.WriteString(inner.Text)
			sb.WriteString(" ")
		}
	}

	return sb.String(), nil
}

func (c *Client) prepareContext(ctx context.Context) (context.Context, error) {
	token, err := c.tokenMgr.GetToken(ctx, auth.ScopeSaluteSpeechPers)
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	return metadata.NewOutgoingContext(ctx, md), nil
}
