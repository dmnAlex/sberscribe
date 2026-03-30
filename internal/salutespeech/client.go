package salutespeech

import (
	"bytes"
	"context"
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
	"google.golang.org/grpc/credentials/insecure"
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
}

func NewSaluteClient(tokenMgr *auth.TokenManager) (*Client, error) {
	conn, err := grpc.NewClient(saluteAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "new grpc client")
	}
	return &Client{tokenMgr: tokenMgr, conn: conn}, nil
}

func (c *Client) Close() { c.conn.Close() }

func (c *Client) Recognize(ctx context.Context, audio []byte, mimeType string) (string, []byte, error) {
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

func (c *Client) upload(ctx context.Context, audio []byte) (string, error) {
	client := storagev1.NewSmartSpeechClient(c.conn)
	stream, err := client.Upload(ctx)
	if err != nil {
		return "", errors.Wrap(err, "upload")
	}

	for i := 0; i < len(audio); i += chunkSize {
		end := i + chunkSize
		if end > len(audio) {
			end = len(audio)
		}

		req := storagev1.UploadRequest_builder{
			FileChunk: audio[i:end],
		}.Build()

		if err := stream.Send(req); err != nil {
			return "", errors.Wrap(err, "send stream")
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		return "", errors.Wrap(err, "close and receive")
	}

	return res.GetRequestFileId(), nil
}

func (c *Client) asyncRecognize(ctx context.Context, requestFileID, mimeType string) (string, error) {
	client := recognitionv1.NewSmartSpeechClient(c.conn)
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

	task, err := client.AsyncRecognize(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "async recognize")
	}

	return task.GetId(), nil
}

func (c *Client) pollTask(ctx context.Context, taskID string) (string, error) {
	client := taskv1.NewSmartSpeechClient(c.conn)
	for i := 0; i < maxPollAttempts; i++ {
		req := taskv1.GetTaskRequest_builder{
			TaskId: taskID,
		}.Build()

		res, err := client.GetTask(ctx, req)
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
	client := storagev1.NewSmartSpeechClient(c.conn)
	req := storagev1.DownloadRequest_builder{
		ResponseFileId: responseFileID,
	}.Build()

	stream, err := client.Download(ctx, req)
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
