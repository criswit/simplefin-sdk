package simplefin

import (
	"net/http"
	"testing"
)

func TestNewClientOptions(t *testing.T) {
	hc := &http.Client{}
	c, err := NewClient("https://u:p@example.com/root", WithHTTPClient(hc))
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient != hc {
		t.Fatal("custom client not used")
	}
	if c.endpoint("/accounts") != "https://example.com/root/accounts" {
		t.Fatalf("endpoint %s", c.endpoint("/accounts"))
	}
}

func TestNewClientDefaultsAndRejectsMalformed(t *testing.T) {
	c, err := NewClient("https://u:p@example.com/root", WithHTTPClient(nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient != http.DefaultClient {
		t.Fatal("nil option should keep default client")
	}
	if _, err := NewClient("http://u:p@example.com/root"); err != ErrInsecureURL {
		t.Fatalf("got %v", err)
	}
	if _, err := NewClient("https://example.com/root"); err != ErrMissingCredentials {
		t.Fatalf("got %v", err)
	}
}
