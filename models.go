package simplefin

import (
	"encoding/json"
	"strings"
	"time"
)

// AccountSet is the SimpleFIN /accounts response.
type AccountSet struct {
	ErrList     []ProtocolError `json:"errlist"`
	Errors      []string        `json:"errors,omitempty"` // deprecated in v2, preserved for compatibility.
	Connections []Connection    `json:"connections"`
	Accounts    []Account       `json:"accounts"`
}

// ProtocolError is a structured SimpleFIN protocol error from errlist.
type ProtocolError struct {
	Code      string `json:"code"`
	Message   string `json:"msg"`
	ConnID    string `json:"conn_id,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Extra     Extra  `json:"-"`
}

// Connection describes a SimpleFIN financial institution connection.
type Connection struct {
	ConnID  string `json:"conn_id"`
	Name    string `json:"name"`
	OrgID   string `json:"org_id"`
	OrgName string `json:"org_name,omitempty"`
	OrgURL  string `json:"org_url,omitempty"`
	SfinURL string `json:"sfin_url"`
	Extra   Extra  `json:"-"`
}

// Account describes a SimpleFIN account.
type Account struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	ConnID           string        `json:"conn_id"`
	Currency         string        `json:"currency"`
	Balance          NumericString `json:"balance"`
	AvailableBalance NumericString `json:"available-balance,omitempty"`
	BalanceDate      UnixTime      `json:"balance-date"`
	Transactions     []Transaction `json:"transactions,omitempty"`
	Extra            Extra         `json:"extra,omitempty"`
}

// Transaction describes a SimpleFIN transaction.
type Transaction struct {
	ID           string        `json:"id"`
	Posted       UnixTime      `json:"posted"`
	Amount       NumericString `json:"amount"`
	Description  string        `json:"description"`
	TransactedAt *UnixTime     `json:"transacted_at,omitempty"`
	Pending      bool          `json:"pending,omitempty"`
	Extra        Extra         `json:"extra,omitempty"`
}

// NumericString preserves protocol numeric values exactly as strings.
type NumericString string

// UnixTime is a Unix timestamp in seconds.
type UnixTime int64

// Extra preserves arbitrary extension JSON fields.
type Extra map[string]json.RawMessage

func (t UnixTime) Time() time.Time { return time.Unix(int64(t), 0) }

func (a Account) BalanceTime() time.Time { return a.BalanceDate.Time() }

func (tx Transaction) PostedTime() time.Time { return tx.Posted.Time() }

// Prefix returns the protocol-error prefix, e.g. gen, con, or act.
func (e ProtocolError) Prefix() string {
	if i := strings.IndexByte(e.Code, '-'); i > 0 {
		return e.Code[:i]
	}
	return e.Code
}

// IsAuth reports whether the protocol error appears to be authentication-related.
func (e ProtocolError) IsAuth() bool {
	code := strings.ToLower(e.Code)
	msg := strings.ToLower(e.Message)
	return strings.Contains(code, "auth") || strings.Contains(code, "login") || strings.Contains(code, "credential") || strings.Contains(msg, "auth") || strings.Contains(msg, "login") || strings.Contains(msg, "credential")
}

func (a AccountSet) HasErrors() bool { return len(a.ErrList) > 0 || len(a.Errors) > 0 }

func (a AccountSet) ProtocolErrors() ProtocolErrors { return ProtocolErrors(a.ErrList) }
