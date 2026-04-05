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
	if cached, ok := m.tokens[scope]; ok && cached.ExpiresAt > time.Now().Unix() {
		m.mu.RUnlock()
		return cached.AccessToken, nil
	}
	m.mu.RUnlock()

	tkn, err := m.oauth.GetToken(ctx, m.creds[scope], scope)
	if err != nil {
		return "", errors.Wrap(err, "oauth get token")
	}
	tkn.ExpiresAt -= int64(tokenMargin.Seconds())

	m.mu.Lock()
	defer m.mu.Unlock()

	if cached, ok := m.tokens[scope]; ok && cached.ExpiresAt > time.Now().Unix() {
		return cached.AccessToken, nil
	}

	m.tokens[scope] = tkn
	return tkn.AccessToken, nil
}
