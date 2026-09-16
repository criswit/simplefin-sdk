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
	pe := ProtocolErrors{{Code: CodeGeneral, Message: "bad"}, {Code: CodeConnectionAuth}}.Error()
	if !strings.Contains(pe, "gen.: bad") || !strings.Contains(pe, "con.auth") {
		t.Fatalf("bad ProtocolErrors: %s", pe)
	}
	if ProtocolErrors(nil).Error() == "" {
		t.Fatal("empty ProtocolErrors should still describe itself")
	}
}
