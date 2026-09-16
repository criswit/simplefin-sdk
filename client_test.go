package simplefin

import (
	"net/http"
	"testing"
)

func TestNewClientOptions(t *testing.T) {
	hc := &http.Client{}
	c, err := NewClient("https://u:p@example.com/root", WithHTTPClient(hc), WithVersion(7))
	if err != nil {
		t.Fatal(err)
	}
	if c.version != 7 {
		t.Fatalf("version %d", c.version)
	}
	if c.httpClient != hc {
		t.Fatal("custom client not used")
	}
}

func TestNewClientDefaultsAndRejectsMalformed(t *testing.T) {
	c, err := NewClient("https://u:p@example.com/root")
	if err != nil {
		t.Fatal(err)
	}
	if c.version != 2 {
		t.Fatalf("version %d", c.version)
	}
	if _, err := NewClient("http://u:p@example.com/root"); err != ErrInsecureURL {
		t.Fatalf("got %v", err)
	}
	if _, err := NewClient("https://example.com/root"); err != ErrMissingCredentials {
		t.Fatalf("got %v", err)
	}
}
