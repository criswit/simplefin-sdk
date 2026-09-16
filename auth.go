package simplefin

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// AccessURL is a parsed SimpleFIN access URL. Raw holds the full URL including
// credentials and must be stored as a secret. Printing an AccessURL with %v or
// %s redacts the password; use Raw when you need the real value.
type AccessURL struct {
	Raw      string
	BaseURL  string
	Username string
	Password string
}

// String returns the access URL with the password redacted.
func (a AccessURL) String() string {
	u, err := url.Parse(a.BaseURL)
	if err != nil || a.BaseURL == "" {
		return "<invalid access URL>"
	}
	u.User = url.UserPassword(a.Username, "REDACTED")
	return u.String()
}

// GoString returns the redacted form for %#v.
func (a AccessURL) GoString() string { return "simplefin.AccessURL{" + a.String() + "}" }

// ParseAccessURL validates an access URL and splits it into its base URL and
// Basic Auth credentials. It returns [ErrInsecureURL] for non-https URLs and
// [ErrMissingCredentials] when the username or password is empty.
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

// ClaimOption configures [ClaimSetupToken].
type ClaimOption func(*claimConfig)

type claimConfig struct{ httpClient *http.Client }

// WithClaimHTTPClient sets the *http.Client used to POST the claim request. A
// nil client is ignored.
func WithClaimHTTPClient(client *http.Client) ClaimOption {
	return func(c *claimConfig) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// ClaimSetupToken exchanges a one-time setup token for an access URL. The
// token is a Base64-encoded https claim URL; the SDK decodes it, POSTs to it
// with an empty body, and parses the returned access URL.
//
// A 403 response yields an error matching [ErrClaimTokenRejected] that also
// wraps *HTTPError. The protocol asks applications to warn the user that the
// token may have been compromised in that case.
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
