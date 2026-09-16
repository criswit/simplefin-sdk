package simplefin

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestModelRoundTripAndHelpers(t *testing.T) {
	transacted := UnixTime(1700000001)
	as := AccountSet{
		ErrList:     []ProtocolError{{Code: "act-auth", Message: "reauth", ConnID: "c", AccountID: "a"}},
		Errors:      []string{"old"},
		Connections: []Connection{{ConnID: "c", Name: "Bank", OrgID: "org", SfinURL: "https://bank.example/sfin"}},
		Accounts:    []Account{{ID: "a", Name: "Acct", ConnID: "c", Currency: "https://currency.example/points", Balance: "1.2300", BalanceDate: UnixTime(1700000000), Transactions: []Transaction{{ID: "t", Posted: UnixTime(1700000000), Amount: "-0.0100", Description: "D", TransactedAt: &transacted, Extra: Extra{"raw": json.RawMessage(`{"a":1}`)}}}, Extra: Extra{"custom": json.RawMessage(`"value"`)}}},
	}
	b, err := json.Marshal(as)
	if err != nil {
		t.Fatal(err)
	}
	var got AccountSet
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Accounts[0].Balance != "1.2300" || got.Accounts[0].Transactions[0].Amount != "-0.0100" {
		t.Fatalf("numeric strings changed: %s", b)
	}
	if string(got.Accounts[0].Extra["custom"]) != `"value"` {
		t.Fatalf("extra lost: %#v", got.Accounts[0].Extra)
	}
	if got.Accounts[0].BalanceTime() != time.Unix(1700000000, 0) || got.Accounts[0].Transactions[0].PostedTime() != time.Unix(1700000000, 0) {
		t.Fatal("time helpers failed")
	}
	if !got.HasErrors() || len(got.ProtocolErrors()) != 1 {
		t.Fatal("error helpers failed")
	}
	pe := got.ErrList[0]
	if pe.Prefix() != "act" || !pe.IsAuth() {
		t.Fatalf("protocol helper failed: %#v", pe)
	}
}

func TestOptionalFieldsOmit(t *testing.T) {
	b, err := json.Marshal(Account{ID: "a", Name: "n", ConnID: "c", Currency: "USD", Balance: "0", BalanceDate: 1})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "transactions") || strings.Contains(s, "available-balance") {
		t.Fatalf("unexpected optional fields: %s", s)
	}
}
