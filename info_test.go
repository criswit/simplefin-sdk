package simplefin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInfo(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simplefin/info" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if _, _, ok := r.BasicAuth(); !ok {
			t.Fatal("expected auth from client")
		}
		_, _ = w.Write([]byte(`{"versions":["1","2"]}`))
	}))
	defer s.Close()
	c, _ := NewClient(testAccessURL(s.URL), WithHTTPClient(s.Client()))
	info, err := c.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Versions) != 2 || info.Versions[1] != "2" {
		t.Fatalf("bad info %#v", info)
	}
}

func TestInfoHTTPError(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusTeapot) }))
	defer s.Close()
	c, _ := NewClient(testAccessURL(s.URL), WithHTTPClient(s.Client()))
	_, err := c.Info(context.Background())
	var herr *HTTPError
	if !errors.As(err, &herr) || herr.StatusCode != http.StatusTeapot {
		t.Fatalf("got %v", err)
	}
}
