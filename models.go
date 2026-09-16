package simplefin

import (
	"encoding/json"
	"strings"
	"time"
)

// AccountSet is the SimpleFIN /accounts response.
type AccountSet struct {
	// ErrList holds structured protocol errors (protocol v2).
	ErrList []ProtocolError `json:"errlist"`
	// Errors holds the deprecated v1 unstructured error strings. The
	// SimpleFIN Bridge still uses this array for rate-limit warnings, so it
	// is preserved.
	Errors      []string     `json:"errors,omitempty"`
	Connections []Connection `json:"connections"`
	Accounts    []Account    `json:"accounts"`
}

// ProtocolError is a structured SimpleFIN protocol error from errlist.
type ProtocolError struct {
	// Code is in the form "prefix.subcode", for example "con.auth". See the
	// Code* constants for the codes defined by the protocol.
	Code string `json:"code"`
	// Message is intended for display to the user. Sanitize it first.
	Message string `json:"msg"`
	// ConnID is set for connection-level ("con.*") errors.
	ConnID string `json:"conn_id,omitempty"`
	// AccountID is set for account-level ("act.*") errors.
	AccountID string `json:"account_id,omitempty"`
}

// Connection describes a single connection to a financial institution. A user
// with two logins at the same bank has two connections that share org_* values.
type Connection struct {
	ConnID string `json:"conn_id"`
	Name   string `json:"name"`
	OrgID  string `json:"org_id"`
	// OrgName is returned by the SimpleFIN Bridge but is not in the protocol's
	// field table.
	OrgName string `json:"org_name,omitempty"`
	OrgURL  string `json:"org_url,omitempty"`
	SfinURL string `json:"sfin_url"`
}

// Account describes a SimpleFIN account.
type Account struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	ConnID string `json:"conn_id"`
	// Currency is an ISO 4217 code or a custom currency URL. See
	// [Client.CurrencyInfo].
	Currency         string        `json:"currency"`
	Balance          NumericString `json:"balance"`
	AvailableBalance NumericString `json:"available-balance,omitempty"`
	BalanceDate      UnixTime      `json:"balance-date"`
	Transactions     []Transaction `json:"transactions,omitempty"`
	// Holdings lists investment positions. It is a SimpleFIN Bridge extension
	// and is absent from the base protocol.
	Holdings []Holding `json:"holdings,omitempty"`
	Extra    Extra     `json:"extra,omitempty"`
}

// Transaction describes a SimpleFIN transaction.
type Transaction struct {
	// ID is unique within the account, not globally. Combine it with the
	// account ID when deduplicating across accounts.
	ID string `json:"id"`
	// Posted is when the transaction posted. It is 0 for pending transactions.
	Posted      UnixTime      `json:"posted"`
	Amount      NumericString `json:"amount"`
	Description string        `json:"description"`
	// Payee, Memo, and MCC are SimpleFIN Bridge extensions. MCC is an ISO 18245
	// merchant category code, sent as a string such as "5411".
	Payee        string    `json:"payee,omitempty"`
	Memo         string    `json:"memo,omitempty"`
	MCC          string    `json:"mcc,omitempty"`
	TransactedAt *UnixTime `json:"transacted_at,omitempty"`
	Pending      bool      `json:"pending,omitempty"`
	Extra        Extra     `json:"extra,omitempty"`
}

// Holding is an investment position reported by the SimpleFIN Bridge.
type Holding struct {
	ID            string        `json:"id"`
	Created       UnixTime      `json:"created"`
	Currency      string        `json:"currency"`
	CostBasis     NumericString `json:"cost_basis"`
	Description   string        `json:"description"`
	MarketValue   NumericString `json:"market_value"`
	PurchasePrice NumericString `json:"purchase_price"`
	Shares        NumericString `json:"shares"`
	Symbol        string        `json:"symbol"`
}

// NumericString preserves protocol numeric values exactly as strings.
type NumericString string

// UnixTime is a Unix timestamp in seconds. Zero means "not set".
type UnixTime int64

// Extra preserves arbitrary extension JSON fields.
type Extra map[string]json.RawMessage

// IsZero reports whether the timestamp is unset.
func (t UnixTime) IsZero() bool { return t == 0 }

// Time converts the timestamp to a time.Time. A zero UnixTime yields the zero
// time.Time rather than the Unix epoch.
func (t UnixTime) Time() time.Time {
	if t == 0 {
		return time.Time{}
	}
	return time.Unix(int64(t), 0)
}

// BalanceTime returns the balance-date as a time.Time.
func (a Account) BalanceTime() time.Time { return a.BalanceDate.Time() }

// IsPosted reports whether the transaction has a posted timestamp. Pending
// transactions have Posted == 0.
func (tx Transaction) IsPosted() bool { return !tx.Posted.IsZero() }

// PostedTime returns the posted timestamp, or the zero time.Time when the
// transaction has not posted.
func (tx Transaction) PostedTime() time.Time { return tx.Posted.Time() }

// TransactedTime returns transacted_at, or the zero time.Time when absent.
func (tx Transaction) TransactedTime() time.Time {
	if tx.TransactedAt == nil {
		return time.Time{}
	}
	return tx.TransactedAt.Time()
}

// EffectiveTime returns the best available timestamp for the transaction:
// posted if set, otherwise transacted_at, otherwise the zero time.Time.
func (tx Transaction) EffectiveTime() time.Time {
	if tx.IsPosted() {
		return tx.PostedTime()
	}
	return tx.TransactedTime()
}

// CreatedTime returns the holding's created timestamp.
func (h Holding) CreatedTime() time.Time { return h.Created.Time() }

// Prefix returns the protocol-error prefix: "gen", "con", or "act". Codes are
// "prefix.subcode", so "con.auth" yields "con". A code without a dot is
// returned unchanged.
func (e ProtocolError) Prefix() string {
	if i := strings.IndexByte(e.Code, '.'); i >= 0 {
		return e.Code[:i]
	}
	return e.Code
}

// Subcode returns the part of the code after the prefix, or "" for a naked
// prefix such as "gen.".
func (e ProtocolError) Subcode() string {
	if i := strings.IndexByte(e.Code, '.'); i >= 0 {
		return e.Code[i+1:]
	}
	return ""
}

// IsKnown reports whether the code is one defined by the protocol.
func (e ProtocolError) IsKnown() bool {
	_, ok := knownCodes[e.Code]
	return ok
}

// Normalized returns the code if the protocol defines it, otherwise the naked
// prefix ("gen.", "con.", or "act."). The protocol asks consumers to treat
// unknown subcodes as the naked prefix.
func (e ProtocolError) Normalized() string {
	if e.IsKnown() {
		return e.Code
	}
	return e.Prefix() + "."
}

// Is reports whether the error matches code. A naked prefix such as
// [CodeConnection] matches every error with that prefix; a full code such as
// [CodeConnectionAuth] matches exactly.
func (e ProtocolError) Is(code string) bool {
	if strings.HasSuffix(code, ".") {
		return e.Prefix()+"." == code
	}
	return e.Code == code
}

// IsAuth reports whether the error is an authentication error, either to the
// SimpleFIN server ([CodeGeneralAuth]) or to a bank connection
// ([CodeConnectionAuth]).
func (e ProtocolError) IsAuth() bool {
	return e.Code == CodeGeneralAuth || e.Code == CodeConnectionAuth
}

// HasErrors reports whether the response carries any structured or legacy
// errors.
func (a AccountSet) HasErrors() bool { return len(a.ErrList) > 0 || len(a.Errors) > 0 }

// ProtocolErrors returns the structured error list as a [ProtocolErrors]
// value, which implements error.
func (a AccountSet) ProtocolErrors() ProtocolErrors { return ProtocolErrors(a.ErrList) }
