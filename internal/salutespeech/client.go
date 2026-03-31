package salutespeech

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/dmnAlex/sberscribe/internal/auth"
	"github.com/dmnAlex/sberscribe/pkg/api/recognitionv1"
	"github.com/dmnAlex/sberscribe/pkg/api/storagev1"
	"github.com/dmnAlex/sberscribe/pkg/api/taskv1"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	saluteAddress   = "smartspeech.sber.ru:443"
	chunkSize       = 64 * 1024
	pollInterval    = 5 * time.Second
	maxPollAttempts = 20
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

func (c *Client) Recognize(ctx context.Context, audio io.Reader, mimeType string) (string, []byte, error) {
	token, err := c.tokenMgr.GetToken(ctx, auth.ScopeSaluteSpeechPers)
	if err != nil {
		return "", nil, errors.Wrap(err, "get token")
	}

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	requestFileID, err := c.upload(ctx, audio)
	if err != nil {
		return "", nil, errors.Wrap(err, "upload")
	}

	taskID, err := c.asyncRecognize(ctx, requestFileID, mimeType)
	if err != nil {
		return "", nil, errors.Wrap(err, "async recognize")
	}

	responceFileID, err := c.pollTask(ctx, taskID)
	if err != nil {
		return "", nil, errors.Wrap(err, "poll task")
	}

	raw, err := c.download(ctx, responceFileID)
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
	encoding := recognitionv1.RecognitionOptions_MP3
	if mimeType == "audio/ogg" {
		encoding = recognitionv1.RecognitionOptions_OPUS
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
	for range maxPollAttempts {
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
			return res.GetResponseFileId(), nil
		case taskv1.Task_ERROR:
			return "", errors.Errorf("recognition error: %s", res.GetError())
		case taskv1.Task_CANCELED:
			return "", errors.New("task canceled")
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return "", errors.New("timeout polling task")
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
