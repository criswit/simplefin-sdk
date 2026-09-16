package simplefin

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// AccountsOptions are the optional query parameters for /accounts.
type AccountsOptions struct {
	// StartDate includes transactions posted on or after this time.
	StartDate *time.Time
	// EndDate includes transactions posted before, but not on, this time.
	EndDate *time.Time
	// Pending asks the server to include pending transactions if it can.
	Pending bool
	// AccountIDs restricts the response to these account IDs.
	AccountIDs []string
	// BalancesOnly omits transaction data.
	BalancesOnly bool
}

// Accounts fetches account, balance, transaction, and holding data from
// /accounts using protocol version 2.
//
// The SimpleFIN Bridge limits date ranges to 90 days and expects roughly 24
// requests per day. The SDK does not enforce those Bridge policies because
// other SimpleFIN servers may differ.
func (c *Client) Accounts(ctx context.Context, opts AccountsOptions) (*AccountSet, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/accounts", nil, true)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("version", ProtocolVersion)
	if opts.StartDate != nil {
		q.Set("start-date", strconv.FormatInt(opts.StartDate.Unix(), 10))
	}
	if opts.EndDate != nil {
		q.Set("end-date", strconv.FormatInt(opts.EndDate.Unix(), 10))
	}
	if opts.Pending {
		q.Set("pending", "1")
	}
	for _, id := range opts.AccountIDs {
		q.Add("account", id)
	}
	if opts.BalancesOnly {
		q.Set("balances-only", "1")
	}
	req.URL.RawQuery = q.Encode()
	var out AccountSet
	if err := c.doJSON(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
