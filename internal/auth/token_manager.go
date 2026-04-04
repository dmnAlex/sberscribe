package auth

import (
	"context"
	"sync"
	"time"

	"github.com/dmnAlex/sberscribe/internal/config"
	"github.com/pkg/errors"
)

const tokenMargin = time.Minute

type OAuthClient interface {
	GetToken(ctx context.Context, clientSecret string, scope Scope) (tokenInfo, error)
}

type TokenManager struct {
	oauth  OAuthClient
	creds  map[Scope]string
	tokens map[Scope]tokenInfo
	mu     sync.RWMutex
}

func NewTokenManager(oauth OAuthClient, cfg *config.Config) *TokenManager {
	return &TokenManager{
		oauth:  oauth,
		tokens: make(map[Scope]tokenInfo),
		creds: map[Scope]string{
			ScopeSaluteSpeechPers: cfg.SaluteSpeechClientSecret,
			ScopeGigaChatPers:     cfg.GigaChatClientSecret,
		},
	}
}

func (m *TokenManager) GetToken(ctx context.Context, scope Scope) (string, error) {
	m.mu.RLock()
	if tkn, ok := m.tokens[scope]; ok && time.Now().Before(time.Unix(tkn.ExpiresAt, 0)) {
		m.mu.RUnlock()
		return tkn.AccessToken, nil
	}
	m.mu.RUnlock()

	tkn, err := m.oauth.GetToken(ctx, m.creds[scope], scope)
	if err != nil {
		return "", errors.Wrap(err, "oauth get token")
	}
	tkn.ExpiresAt = time.Unix(tkn.ExpiresAt, 0).Add(-tokenMargin).Unix()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens[scope] = tkn

	return tkn.AccessToken, nil
}
