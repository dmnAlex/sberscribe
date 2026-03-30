package auth

import (
	"context"
	"sync"
	"time"

	"github.com/dmnAlex/sberscribe/internal/config"
	"github.com/pkg/errors"
)

const tokenDuration = 25 * time.Minute

type OAuthClient interface {
	GetToken(ctx context.Context, clientSecret string, scope Scope) (string, error)
}

type tokenEntry struct {
	token     string
	expiresAt time.Time
}

type TokenManager struct {
	oauth  OAuthClient
	creds  map[Scope]string
	mu     sync.RWMutex
	tokens map[Scope]tokenEntry
}

func NewTokenManager(oauth OAuthClient, cfg *config.Config) *TokenManager {
	return &TokenManager{
		oauth:  oauth,
		tokens: make(map[Scope]tokenEntry),
		creds: map[Scope]string{
			ScopeSaluteSpeechPers: cfg.SaluteSpeechClientSecret,
			ScopeGigaChatPers:     cfg.GigaChatClientSecret,
		},
	}
}

func (m *TokenManager) GetToken(ctx context.Context, scope Scope) (string, error) {
	m.mu.RLock()
	if entry, ok := m.tokens[scope]; ok && time.Now().Before(entry.expiresAt) {
		m.mu.RUnlock()
		return entry.token, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	token, err := m.oauth.GetToken(ctx, m.creds[scope], scope)
	if err != nil {
		return "", errors.Wrap(err, "oauth get token")
	}

	m.tokens[scope] = tokenEntry{
		token:     token,
		expiresAt: time.Now().Add(tokenDuration),
	}

	return token, nil
}
