package simplefin

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type AccountsOptions struct {
	StartDate    *time.Time
	EndDate      *time.Time
	Pending      bool
	AccountIDs   []string
	BalancesOnly bool
	Version      int
}

// Accounts fetches account, balance, and transaction data. Some SimpleFIN Bridge
// servers limit date ranges to 90 days; the SDK does not enforce that
// bridge-specific policy.
func (c *Client) Accounts(ctx context.Context, opts AccountsOptions) (*AccountSet, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/accounts", nil, true)
	if err != nil {
		return nil, err
	}
	version := opts.Version
	if version == 0 {
		version = c.version
	}
	q := req.URL.Query()
	q.Set("version", strconv.Itoa(version))
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
