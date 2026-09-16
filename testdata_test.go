package simplefin

import (
	"encoding/json"
	"os"
	"testing"
)

// TestBridgeFixture decodes a real SimpleFIN Bridge v2 response captured from
// the public demo server and checks that Bridge extension fields survive.
func TestBridgeFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/accounts_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var as AccountSet
	if err := json.Unmarshal(raw, &as); err != nil {
		t.Fatal(err)
	}
	if as.HasErrors() {
		t.Fatalf("fixture has errors: %#v", as.ErrList)
	}
	if len(as.Connections) != 1 || as.Connections[0].ConnID != "CON-SIMPLEFIN-DEMO" || as.Connections[0].OrgName != "SimpleFIN Bridge" {
		t.Fatalf("connections: %#v", as.Connections)
	}
	if len(as.Accounts) < 2 {
		t.Fatalf("expected demo accounts, got %d", len(as.Accounts))
	}
	var sawTx, sawHolding bool
	for _, a := range as.Accounts {
		if a.ConnID != "CON-SIMPLEFIN-DEMO" || a.Currency != "USD" || a.Balance == "" || a.BalanceDate == 0 {
			t.Fatalf("account fields: %#v", a)
		}
		for _, tx := range a.Transactions {
			sawTx = true
			if tx.Payee == "" || tx.Memo == "" || tx.MCC == "" || !tx.IsPosted() || tx.TransactedAt == nil {
				t.Fatalf("transaction lost bridge fields: %#v", tx)
			}
		}
		for _, h := range a.Holdings {
			sawHolding = true
			if h.ID == "" || h.Symbol == "" || h.Shares == "" || h.MarketValue == "" || h.CostBasis == "" || h.Currency == "" || h.Created == 0 {
				t.Fatalf("holding lost fields: %#v", h)
			}
		}
	}
	if !sawTx || !sawHolding {
		t.Fatalf("fixture should contain transactions (%v) and holdings (%v)", sawTx, sawHolding)
	}
}
