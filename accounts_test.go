package simplefin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testAccessURL(serverURL string) string {
	return "https://user:pass@" + serverURL[len("https://"):] + "/simplefin"
}

func TestAccountsRequestAndResponse(t *testing.T) {
	start := time.Unix(100, 0)
	end := time.Unix(200, 0)
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simplefin/accounts" {
			t.Fatalf("path %s", r.URL.Path)
		}
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pass" {
			t.Fatalf("bad auth")
		}
		q := r.URL.Query()
		want := map[string]string{"version": "2", "balances-only": "1", "pending": "1", "start-date": "100", "end-date": "200"}
		for k, v := range want {
			if q.Get(k) != v {
				t.Fatalf("%s=%q want %q; raw %s", k, q.Get(k), v, r.URL.RawQuery)
			}
		}
		accounts := q["account"]
		if len(accounts) != 2 || accounts[0] != "a1" || accounts[1] != "a2" {
			t.Fatalf("accounts %#v", accounts)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"errlist":[{"code":"act-auth","msg":"reauth","conn_id":"c1","account_id":"a1"}],
			"errors":["deprecated"],
			"connections":[{"conn_id":"c1","name":"Conn","org_id":"org","org_name":"Org","org_url":"https://org.example","sfin_url":"https://sfin.example"}],
			"accounts":[{"id":"a1","name":"Checking","conn_id":"c1","currency":"USD","balance":"123.4500","available-balance":"120.00","balance-date":100,"transactions":[{"id":"t1","posted":90,"amount":"-1.25","description":"Coffee","transacted_at":80,"pending":true,"extra":{"k":{"nested":1}}}],"extra":{"x":"y"}}]
		}`))
	}))
	defer s.Close()
	c, err := NewClient(testAccessURL(s.URL), WithHTTPClient(s.Client()))
	if err != nil {
		t.Fatal(err)
	}
	as, err := c.Accounts(context.Background(), AccountsOptions{StartDate: &start, EndDate: &end, Pending: true, AccountIDs: []string{"a1", "a2"}, BalancesOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if !as.HasErrors() || len(as.ErrList) != 1 || len(as.Errors) != 1 {
		t.Fatalf("errors not preserved: %#v", as)
	}
	if as.Accounts[0].Balance != NumericString("123.4500") {
		t.Fatalf("balance changed: %q", as.Accounts[0].Balance)
	}
	if as.Accounts[0].Transactions[0].TransactedAt == nil || !as.Accounts[0].Transactions[0].Pending {
		t.Fatal("transaction optionals missing")
	}
}

func TestAccountsStatusErrorsAndDecode(t *testing.T) {
	for status, sentinel := range map[int]error{http.StatusPaymentRequired: ErrPaymentRequired, http.StatusForbidden: ErrAuthenticationFailed} {
		s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "bad", status) }))
		c, _ := NewClient(testAccessURL(s.URL), WithHTTPClient(s.Client()))
		_, err := c.Accounts(context.Background(), AccountsOptions{})
		if !errors.Is(err, sentinel) {
			t.Fatalf("status %d got %v", status, err)
		}
		var herr *HTTPError
		if !errors.As(err, &herr) {
			t.Fatalf("missing HTTPError")
		}
		s.Close()
	}
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{`)) }))
	defer s.Close()
	c, _ := NewClient(testAccessURL(s.URL), WithHTTPClient(s.Client()))
	if _, err := c.Accounts(context.Background(), AccountsOptions{}); err == nil {
		t.Fatal("expected decode error")
	}
}
