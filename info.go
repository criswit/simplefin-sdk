package simplefin

import (
	"context"
	"net/http"
)

type InfoResponse struct {
	Versions []string `json:"versions"`
}

// Info fetches the server's protocol information. For an authenticated Client,
// Basic Auth is sent consistently with other client requests.
func (c *Client) Info(ctx context.Context) (*InfoResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/info", nil, true)
	if err != nil {
		return nil, err
	}
	var out InfoResponse
	if err := c.doJSON(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
