package simplefin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ProtocolVersion is the SimpleFIN protocol version this package speaks. It is
// sent as the version query parameter on every /accounts request.
const ProtocolVersion = "2"

// Client is a SimpleFIN protocol client bound to one access URL.
type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the *http.Client used for requests. Use it to configure
// timeouts, transports, proxies, or test servers. A nil client is ignored.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// NewClient creates a Client from an access URL such as
// "https://user:pass@bridge.simplefin.org/simplefin". The URL must be https
// and must carry a username and password, which are used for HTTP Basic Auth.
func NewClient(accessURL string, opts ...Option) (*Client, error) {
	parsed, err := ParseAccessURL(accessURL)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(parsed.BaseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{baseURL: base, username: parsed.Username, password: parsed.Password, httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Client) endpoint(path string) string {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader, auth bool) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), body)
	if err != nil {
		return nil, err
	}
	if auth {
		req.SetBasicAuth(c.username, c.password)
	}
	return req, nil
}

func (c *Client) doJSON(req *http.Request, dst any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		herr := &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: body}
		switch resp.StatusCode {
		case http.StatusPaymentRequired:
			return wrapStatus(ErrPaymentRequired, herr)
		case http.StatusForbidden:
			return wrapStatus(ErrAuthenticationFailed, herr)
		default:
			return herr
		}
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
