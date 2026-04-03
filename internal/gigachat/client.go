package gigachat

import (
	"context"
	"crypto/tls"

	"github.com/dmnAlex/sberscribe/internal/auth"
	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/pkg/api/gigachatv1"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	gigaAddr = "gigachat.devices.sberbank.ru:443"
)

type Client struct {
	tokenMgr *auth.TokenManager
	conn     *grpc.ClientConn

	model       string
	chatClient  gigachatv1.ChatServiceClient
	modelClient gigachatv1.ModelsServiceClient
}

type ChatModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
	Type    string `json:"type"`
}

func NewGigaClient(tokenMgr *auth.TokenManager, tlsConfig *tls.Config, model string) (*Client, error) {
	conn, err := grpc.NewClient(gigaAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, errors.Wrap(err, "new grpc client")
	}

	logger.Log.Debug("using chat model", "model", model)
	return &Client{
		tokenMgr:    tokenMgr,
		conn:        conn,
		model:       model,
		chatClient:  gigachatv1.NewChatServiceClient(conn),
		modelClient: gigachatv1.NewModelsServiceClient(conn),
	}, nil
}

func (c *Client) Close() { c.conn.Close() }

func (c *Client) Chat(ctx context.Context, msgs []model.ChatMessage) (string, error) {
	ctx, err := c.prepareContext(ctx)
	if err != nil {
		return "", errors.Wrap(err, "prepare context")
	}

	gigaMsgs := make([]*gigachatv1.Message, 0, len(msgs))
	for i := range msgs {
		gigaMsgs = append(gigaMsgs, gigachatv1.Message_builder{
			Role:    msgs[i].Role.String(),
			Content: msgs[i].Content,
		}.Build())
	}

	req := gigachatv1.ChatRequest_builder{
		Model:    c.model,
		Messages: gigaMsgs,
	}.Build()

	res, err := c.chatClient.Chat(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "chat")
	}

	return res.GetAlternatives()[0].GetMessage().GetContent(), nil
}

func (c *Client) GetModels(ctx context.Context) ([]ChatModel, error) {
	ctx, err := c.prepareContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "prepare context")
	}

	req := gigachatv1.ListModelsRequest_builder{}.Build()

	res, err := c.modelClient.ListModels(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "list models")
	}

	var models []ChatModel
	for _, model := range res.GetModels() {
		models = append(models, ChatModel{
			ID:      model.GetName(),
			Object:  model.GetObject(),
			OwnedBy: model.GetOwnedBy(),
			Type:    model.GetType(),
		})
	}

	return models, nil
}

func (c *Client) prepareContext(ctx context.Context) (context.Context, error) {
	token, err := c.tokenMgr.GetToken(ctx, auth.ScopeGigaChatPers)
	if err != nil {
		return nil, errors.Wrap(err, "get token")
	}

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	return metadata.NewOutgoingContext(ctx, md), nil
}
