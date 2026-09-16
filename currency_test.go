package simplefin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrencyInfo(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"Reward Points","abbr":"PTS"}`))
	}))
	defer s.Close()
	c, _ := NewClient("https://u:p@example.com/simplefin", WithHTTPClient(s.Client()))
	ci, err := c.CurrencyInfo(context.Background(), s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if ci.Name != "Reward Points" || ci.Abbr != "PTS" {
		t.Fatalf("bad currency %#v", ci)
	}
}

func TestCurrencyInfoErrors(t *testing.T) {
	c, _ := NewClient("https://u:p@example.com/simplefin")
	if _, err := c.CurrencyInfo(context.Background(), "http://example.com/currency"); !errors.Is(err, ErrInsecureURL) {
		t.Fatalf("got %v", err)
	}

	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusBadGateway) }))
	defer s.Close()
	c, _ = NewClient("https://u:p@example.com/simplefin", WithHTTPClient(s.Client()))
	_, err := c.CurrencyInfo(context.Background(), s.URL)
	var herr *HTTPError
	if !errors.As(err, &herr) || herr.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %v", err)
	}

	s2 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{`)) }))
	defer s2.Close()
	c, _ = NewClient("https://u:p@example.com/simplefin", WithHTTPClient(s2.Client()))
	if _, err := c.CurrencyInfo(context.Background(), s2.URL); err == nil {
		t.Fatal("expected malformed JSON error")
	}
}
