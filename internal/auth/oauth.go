package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	defaultAuthURL    = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	httpClientTimeout = 10 * time.Second
)

type Scope string

const (
	ScopeSaluteSpeechPers Scope = "SALUTE_SPEECH_PERS"
	ScopeGigaChatPers     Scope = "GIGACHAT_API_PERS"
)

func (s Scope) String() string {
	return string(s)
}

type tokenInfo struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type OAuthHTTPClient struct {
	httpClient *http.Client
	authURL    string
}

func NewOAuthHTTPClient(tlsConfig *tls.Config) *OAuthHTTPClient {
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &OAuthHTTPClient{
		httpClient: &http.Client{
			Timeout:   httpClientTimeout,
			Transport: transport,
		},
		authURL: defaultAuthURL,
	}
}

func (c *OAuthHTTPClient) GetToken(ctx context.Context, clientSecret string, scope Scope) (tokenInfo, error) {
	data := url.Values{
		"scope": {scope.String()},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return tokenInfo{}, errors.Wrap(err, "new request with context")
	}
	req.Header.Set("Authorization", "Basic "+clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("RqUID", uuid.NewString())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return tokenInfo{}, errors.Wrap(err, "http do")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return tokenInfo{}, errors.Errorf("oauth bad status: %d", res.StatusCode)
	}

	var tr tokenInfo
	if err := json.NewDecoder(res.Body).Decode(&tr); err != nil {
		return tokenInfo{}, errors.Wrap(err, "json decode")
	}
	if tr.AccessToken == "" {
		return tokenInfo{}, errors.New("empty access_token")
	}

	logger.Log.Debug("got new token", "scope", scope, "exp", tr.ExpiresAt)

	return tr, nil
}
