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
	GetToken(ctx context.Context, clientSecret string, scope Scope) (tokenResponse, error)
}

type TokenManager struct {
	oauth  OAuthClient
	creds  map[Scope]string
	tokens map[Scope]tokenResponse
	mu     sync.RWMutex
}

func NewTokenManager(oauth OAuthClient, cfg *config.Config) *TokenManager {
	return &TokenManager{
		oauth:  oauth,
		tokens: make(map[Scope]tokenResponse),
		creds: map[Scope]string{
			ScopeSaluteSpeechPers: cfg.SaluteSpeechClientSecret,
			ScopeGigaChatPers:     cfg.GigaChatClientSecret,
		},
	}
}

func (m *TokenManager) GetToken(ctx context.Context, scope Scope) (string, error) {
	m.mu.RLock()
	if entry, ok := m.tokens[scope]; ok && time.Now().Before(time.Unix(entry.ExpiresAt, 0)) {
		m.mu.RUnlock()
		return entry.AccessToken, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	tr, err := m.oauth.GetToken(ctx, m.creds[scope], scope)
	if err != nil {
		return "", errors.Wrap(err, "oauth get token")
	}
	tr.ExpiresAt = time.Unix(tr.ExpiresAt, 0).Add(-tokenMargin).Unix()
	m.tokens[scope] = tr

	return tr.AccessToken, nil
}
