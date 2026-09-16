package simplefin

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseAccessURL(t *testing.T) {
	a, err := ParseAccessURL("https://u%40x:p%2Fq@example.com/simplefin")
	if err != nil {
		t.Fatal(err)
	}
	if a.Username != "u@x" || a.Password != "p/q" {
		t.Fatalf("bad creds: %#v", a)
	}
	if a.BaseURL != "https://example.com/simplefin" {
		t.Fatalf("base contains creds or wrong: %s", a.BaseURL)
	}
}

func TestAccessURLRedactsPassword(t *testing.T) {
	a, err := ParseAccessURL("https://demo:secret@example.com/simplefin")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{a.String(), a.GoString(), fmt.Sprint(a), fmt.Sprintf("%v", a), fmt.Sprintf("%+v", a), fmt.Sprintf("%#v", a), fmt.Sprintf("%s", a)} {
		if strings.Contains(s, "secret") {
			t.Fatalf("password leaked: %s", s)
		}
		if !strings.Contains(s, "demo") || !strings.Contains(s, "example.com") {
			t.Fatalf("redacted form lost host or user: %s", s)
		}
	}
	if a.Raw != "https://demo:secret@example.com/simplefin" {
		t.Fatal("Raw must keep the full credential")
	}
	if (AccessURL{}).String() == "" {
		t.Fatal("zero AccessURL should still print")
	}
}

func TestParseAccessURLRejectsInvalid(t *testing.T) {
	cases := []struct {
		raw  string
		want error
	}{
		{"http://u:p@example.com/simplefin", ErrInsecureURL},
		{"https://example.com/simplefin", ErrMissingCredentials},
		{"https://u@example.com/simplefin", ErrMissingCredentials},
		{"https://u:@example.com/simplefin", ErrMissingCredentials},
	}
	for _, tc := range cases {
		_, err := ParseAccessURL(tc.raw)
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: got %v want %v", tc.raw, err, tc.want)
		}
	}
}

func TestClaimSetupToken(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		if r.ContentLength > 0 {
			t.Fatalf("content length %d", r.ContentLength)
		}
		_, _ = w.Write([]byte("https://user:pass@example.com/simplefin"))
	}))
	defer s.Close()
	tok := base64.StdEncoding.EncodeToString([]byte(s.URL))
	a, err := ClaimSetupToken(context.Background(), tok, WithClaimHTTPClient(s.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if a.Username != "user" || a.Password != "pass" {
		t.Fatalf("bad access URL: %#v", a)
	}
}

func TestClaimSetupTokenErrors(t *testing.T) {
	if _, err := ClaimSetupToken(context.Background(), "not base64"); err == nil {
		t.Fatal("expected invalid base64 error")
	}
	insecure := base64.StdEncoding.EncodeToString([]byte("http://example.com/claim"))
	if _, err := ClaimSetupToken(context.Background(), insecure); !errors.Is(err, ErrInsecureURL) {
		t.Fatalf("got %v", err)
	}

	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusForbidden) }))
	defer s.Close()
	_, err := ClaimSetupToken(context.Background(), base64.StdEncoding.EncodeToString([]byte(s.URL)), WithClaimHTTPClient(s.Client()))
	if !errors.Is(err, ErrClaimTokenRejected) {
		t.Fatalf("got %v", err)
	}
	var herr *HTTPError
	if !errors.As(err, &herr) {
		t.Fatalf("expected HTTPError in chain: %v", err)
	}
}
