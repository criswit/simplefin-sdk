package simplefin

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInsecureURL = errors.New("simplefin: insecure URL")
var ErrMissingCredentials = errors.New("simplefin: access URL missing credentials")
var ErrClaimTokenRejected = errors.New("simplefin: setup token rejected or already claimed")
var ErrPaymentRequired = errors.New("simplefin: payment required")
var ErrAuthenticationFailed = errors.New("simplefin: authentication failed")

// HTTPError represents a non-2xx HTTP response.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if len(e.Body) == 0 {
		return fmt.Sprintf("simplefin: http error: %s", e.Status)
	}
	return fmt.Sprintf("simplefin: http error: %s: %s", e.Status, strings.TrimSpace(string(e.Body)))
}

// ProtocolErrors is a list of SimpleFIN protocol errors.
type ProtocolErrors []ProtocolError

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
