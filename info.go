package simplefin

import (
	"context"
	"net/http"
)

// InfoResponse is the /info response.
type InfoResponse struct {
	// Versions lists supported protocol versions as "MAJOR.MINOR" or
	// "MAJOR.MINOR.FIX" strings. Note that the SimpleFIN Bridge reports only
	// "1.0" here even though it serves version 2, so this cannot be used to
	// detect v2 support.
	Versions []string `json:"versions"`
}

// Info fetches the server's protocol information from /info. Basic Auth is
// sent, consistent with the other client requests.
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
