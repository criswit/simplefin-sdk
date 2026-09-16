package simplefin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// CurrencyInfo describes a custom (non ISO 4217) currency such as reward
// points or airline miles.
type CurrencyInfo struct {
	Name string `json:"name"`
	Abbr string `json:"abbr"`
}

// CurrencyInfo fetches metadata for a custom currency URL found in
// [Account.Currency]. The URL must be https. No credentials are sent. Callers
// should sanitize the returned strings before displaying them.
func (c *Client) CurrencyInfo(ctx context.Context, currencyURL string) (*CurrencyInfo, error) {
	u, err := url.Parse(currencyURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" {
		return nil, ErrInsecureURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, currencyURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: body}
	}
	var out CurrencyInfo
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
