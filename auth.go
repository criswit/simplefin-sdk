package simplefin

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type AccessURL struct {
	Raw      string
	BaseURL  string
	Username string
	Password string
}

func ParseAccessURL(raw string) (AccessURL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return AccessURL{}, err
	}
	if u.Scheme != "https" {
		return AccessURL{}, ErrInsecureURL
	}
	if u.User == nil {
		return AccessURL{}, ErrMissingCredentials
	}
	username := u.User.Username()
	password, hasPassword := u.User.Password()
	if username == "" || !hasPassword || password == "" {
		return AccessURL{}, ErrMissingCredentials
	}
	base := *u
	base.User = nil
	return AccessURL{Raw: raw, BaseURL: base.String(), Username: username, Password: password}, nil
}

type ClaimOption func(*claimConfig)

type claimConfig struct{ httpClient *http.Client }

func WithClaimHTTPClient(client *http.Client) ClaimOption {
	return func(c *claimConfig) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func ClaimSetupToken(ctx context.Context, setupToken string, opts ...ClaimOption) (AccessURL, error) {
	claimURLBytes, err := base64.StdEncoding.DecodeString(setupToken)
	if err != nil {
		return AccessURL{}, err
	}
	claimURL := string(claimURLBytes)
	u, err := url.Parse(claimURL)
	if err != nil {
		return AccessURL{}, err
	}
	if u.Scheme != "https" {
		return AccessURL{}, ErrInsecureURL
	}
	cfg := claimConfig{httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(&cfg)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claimURL, nil)
	if err != nil {
		return AccessURL{}, err
	}
	resp, err := cfg.httpClient.Do(req)
	if err != nil {
		return AccessURL{}, err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		herr := &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: body}
		if resp.StatusCode == http.StatusForbidden {
			return AccessURL{}, wrapStatus(ErrClaimTokenRejected, herr)
		}
		return AccessURL{}, herr
	}
	if readErr != nil {
		return AccessURL{}, readErr
	}
	accessURL, err := ParseAccessURL(string(body))
	if err != nil {
		return AccessURL{}, fmt.Errorf("simplefin: invalid claimed access URL: %w", err)
	}
	return accessURL, nil
}
