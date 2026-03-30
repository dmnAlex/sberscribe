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

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type OAuthHTTPClient struct {
	httpClient *http.Client
	authURL    string
}

func NewOAuthHTTPClient() *OAuthHTTPClient {
	// TODO добавить сертификаты
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &OAuthHTTPClient{
		httpClient: &http.Client{
			Timeout:   httpClientTimeout,
			Transport: transport,
		},
		authURL: defaultAuthURL,
	}
}

func (c *OAuthHTTPClient) GetToken(ctx context.Context, clientSecret string, scope Scope) (string, error) {
	data := url.Values{
		"scope": {scope.String()},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", errors.Wrap(err, "new request with context")
	}
	req.Header.Set("Authorization", "Basic "+clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("RqUID", uuid.NewString())

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "http do")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", errors.Errorf("oauth bad status: %d", res.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tr); err != nil {
		return "", errors.Wrap(err, "json decode")
	}
	if tr.AccessToken == "" {
		return "", errors.New("empty access_token")
	}

	logger.Log.Debug("got new token", "scope", scope, "token", tr.AccessToken, "exp", tr.ExpiresIn) // TODO DEBUG

	return tr.AccessToken, nil
}
