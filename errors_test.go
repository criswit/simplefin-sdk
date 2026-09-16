package simplefin

import (
	"strings"
	"testing"
)

func TestErrorStrings(t *testing.T) {
	he := (&HTTPError{StatusCode: 500, Status: "500 Internal Server Error", Body: []byte(" boom \n")}).Error()
	if !strings.Contains(he, "500 Internal Server Error") || !strings.Contains(he, "boom") {
		t.Fatalf("bad HTTPError: %s", he)
	}
	pe := ProtocolErrors{{Code: "gen-error", Message: "bad"}, {Code: "act-auth"}}.Error()
	if !strings.Contains(pe, "gen-error: bad") || !strings.Contains(pe, "act-auth") {
		t.Fatalf("bad ProtocolErrors: %s", pe)
	}
}
