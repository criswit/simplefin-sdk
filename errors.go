package simplefin

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors. Status sentinels wrap a *HTTPError, so both errors.Is on
// the sentinel and errors.As on *HTTPError work.
var (
	// ErrInsecureURL is returned when a claim, access, or currency URL is not
	// https.
	ErrInsecureURL = errors.New("simplefin: insecure URL")
	// ErrMissingCredentials is returned when an access URL has no username or
	// password.
	ErrMissingCredentials = errors.New("simplefin: access URL missing credentials")
	// ErrClaimTokenRejected is returned when the claim endpoint answers 403,
	// meaning the setup token is invalid or was already claimed. The protocol
	// asks applications to warn the user that the token may be compromised.
	ErrClaimTokenRejected = errors.New("simplefin: setup token rejected or already claimed")
	// ErrPaymentRequired is returned when /accounts answers 402.
	ErrPaymentRequired = errors.New("simplefin: payment required")
	// ErrAuthenticationFailed is returned when the server answers 403 to an
	// authenticated request, meaning the access URL is invalid or revoked.
	ErrAuthenticationFailed = errors.New("simplefin: authentication failed")
)

// Protocol error prefixes. See [ProtocolError.Prefix].
const (
	PrefixGeneral    = "gen"
	PrefixConnection = "con"
	PrefixAccount    = "act"
)

// Protocol error codes defined by SimpleFIN v2. Servers may return other
// subcodes; use [ProtocolError.Normalized] to fall back to the naked prefix.
const (
	// CodeGeneral is a general error with no subcode.
	CodeGeneral = "gen."
	// CodeGeneralAPI means the API is being misused. It is aimed at the
	// developer, not the user.
	CodeGeneralAPI = "gen.api"
	// CodeGeneralAuth is an authentication failure to the SimpleFIN server.
	CodeGeneralAuth = "gen.auth"
	// CodeConnection is a general connection-level error (conn_id set).
	CodeConnection = "con."
	// CodeConnectionAuth means the bank connection needs re-authentication.
	CodeConnectionAuth = "con.auth"
	// CodeAccount is a general account-level error (account_id set).
	CodeAccount = "act."
	// CodeAccountFailed means the account could not be fetched. Retry later.
	CodeAccountFailed = "act.failed"
	// CodeAccountMissingData means the transaction listing is incomplete.
	// Retry later.
	CodeAccountMissingData = "act.missingdata"
)

var knownCodes = map[string]struct{}{
	CodeGeneral: {}, CodeGeneralAPI: {}, CodeGeneralAuth: {},
	CodeConnection: {}, CodeConnectionAuth: {},
	CodeAccount: {}, CodeAccountFailed: {}, CodeAccountMissingData: {},
}

// HTTPError represents a non-2xx HTTP response.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

// Error implements error.
func (e *HTTPError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if len(e.Body) == 0 {
		return fmt.Sprintf("simplefin: http error: %s", e.Status)
	}
	return fmt.Sprintf("simplefin: http error: %s: %s", e.Status, strings.TrimSpace(string(e.Body)))
}

// ProtocolErrors is a list of SimpleFIN protocol errors. It implements error
// so callers can surface a non-empty errlist as a Go error.
type ProtocolErrors []ProtocolError

// Error implements error.
func (e ProtocolErrors) Error() string {
	if len(e) == 0 {
		return "simplefin: protocol errors"
	}
	parts := make([]string, 0, len(e))
	for _, pe := range e {
		if pe.Message != "" {
			parts = append(parts, pe.Code+": "+pe.Message)
		} else {
			parts = append(parts, pe.Code)
		}
	}
	return "simplefin: protocol errors: " + strings.Join(parts, "; ")
}

type statusError struct {
	sentinel error
	httpErr  *HTTPError
}

func (e *statusError) Error() string        { return e.sentinel.Error() + ": " + e.httpErr.Error() }
func (e *statusError) Unwrap() error        { return e.httpErr }
func (e *statusError) Is(target error) bool { return target == e.sentinel }

func wrapStatus(sentinel error, httpErr *HTTPError) error {
	return &statusError{sentinel: sentinel, httpErr: httpErr}
}
