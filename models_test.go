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
		ErrList:     []ProtocolError{{Code: CodeConnectionAuth, Message: "reauth", ConnID: "c"}},
		Errors:      []string{"old"},
		Connections: []Connection{{ConnID: "c", Name: "Bank", OrgID: "org", SfinURL: "https://bank.example/sfin"}},
		Accounts: []Account{{
			ID: "a", Name: "Acct", ConnID: "c", Currency: "https://currency.example/points", Balance: "1.2300", BalanceDate: UnixTime(1700000000),
			Transactions: []Transaction{{ID: "t", Posted: UnixTime(1700000000), Amount: "-0.0100", Description: "D", Payee: "P", Memo: "M", MCC: "5411", TransactedAt: &transacted, Extra: Extra{"raw": json.RawMessage(`{"a":1}`)}}},
			Holdings:     []Holding{{ID: "h", Created: 1, Currency: "USD", CostBasis: "55.00", Description: "Apple", MarketValue: "100.5", PurchasePrice: "0.10", Shares: "550.0", Symbol: "AAPL"}},
			Extra:        Extra{"custom": json.RawMessage(`"value"`)},
		}},
	}
	b, err := json.Marshal(as)
	if err != nil {
		t.Fatal(err)
	}
	var got AccountSet
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	acct := got.Accounts[0]
	tx := acct.Transactions[0]
	if acct.Balance != "1.2300" || tx.Amount != "-0.0100" {
		t.Fatalf("numeric strings changed: %s", b)
	}
	if tx.Payee != "P" || tx.Memo != "M" || tx.MCC != "5411" {
		t.Fatalf("bridge transaction fields lost: %#v", tx)
	}
	if len(acct.Holdings) != 1 || acct.Holdings[0].Symbol != "AAPL" || acct.Holdings[0].MarketValue != "100.5" || acct.Holdings[0].CreatedTime() != time.Unix(1, 0) {
		t.Fatalf("holdings lost: %#v", acct.Holdings)
	}
	if string(acct.Extra["custom"]) != `"value"` {
		t.Fatalf("extra lost: %#v", acct.Extra)
	}
	if acct.BalanceTime() != time.Unix(1700000000, 0) || tx.PostedTime() != time.Unix(1700000000, 0) || tx.TransactedTime() != time.Unix(1700000001, 0) {
		t.Fatal("time helpers failed")
	}
	if !got.HasErrors() || len(got.ProtocolErrors()) != 1 {
		t.Fatal("error helpers failed")
	}
	pe := got.ErrList[0]
	if pe.Prefix() != PrefixConnection || !pe.IsAuth() {
		t.Fatalf("protocol helper failed: %#v", pe)
	}
}

func TestOptionalFieldsOmit(t *testing.T) {
	b, err := json.Marshal(Account{ID: "a", Name: "n", ConnID: "c", Currency: "USD", Balance: "0", BalanceDate: 1})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, f := range []string{"transactions", "available-balance", "holdings", "extra"} {
		if strings.Contains(s, f) {
			t.Fatalf("unexpected optional field %s: %s", f, s)
		}
	}
	b, _ = json.Marshal(Transaction{ID: "t", Amount: "1", Description: "d"})
	for _, f := range []string{"payee", "memo", "mcc", "transacted_at", "pending"} {
		if strings.Contains(string(b), f) {
			t.Fatalf("unexpected optional field %s: %s", f, b)
		}
	}
}

func TestPendingTransactionTimes(t *testing.T) {
	pending := Transaction{ID: "p", Posted: 0, Pending: true}
	if pending.IsPosted() || !pending.PostedTime().IsZero() || !pending.EffectiveTime().IsZero() {
		t.Fatalf("pending transaction should have zero times: %v", pending.PostedTime())
	}
	at := UnixTime(1700000005)
	pending.TransactedAt = &at
	if pending.EffectiveTime() != time.Unix(1700000005, 0) {
		t.Fatalf("EffectiveTime should fall back to transacted_at: %v", pending.EffectiveTime())
	}
	posted := Transaction{ID: "x", Posted: 1700000000, TransactedAt: &at}
	if !posted.IsPosted() || posted.EffectiveTime() != time.Unix(1700000000, 0) {
		t.Fatal("EffectiveTime should prefer posted")
	}
	if !UnixTime(0).IsZero() || !UnixTime(0).Time().IsZero() {
		t.Fatal("zero UnixTime should map to zero time.Time")
	}
}

func TestProtocolErrorCodes(t *testing.T) {
	cases := []struct {
		code                        string
		prefix, subcode, normalized string
		known, auth                 bool
	}{
		{"gen.api", "gen", "api", "gen.api", true, false},
		{"gen.auth", "gen", "auth", "gen.auth", true, true},
		{"con.auth", "con", "auth", "con.auth", true, true},
		{"con.", "con", "", "con.", true, false},
		{"act.failed", "act", "failed", "act.failed", true, false},
		{"act.missingdata", "act", "missingdata", "act.missingdata", true, false},
		{"act.somethingnew", "act", "somethingnew", "act.", false, false},
		{"weird", "weird", "", "weird.", false, false},
	}
	for _, tc := range cases {
		pe := ProtocolError{Code: tc.code}
		if pe.Prefix() != tc.prefix || pe.Subcode() != tc.subcode || pe.Normalized() != tc.normalized || pe.IsKnown() != tc.known || pe.IsAuth() != tc.auth {
			t.Errorf("%s: prefix=%q subcode=%q normalized=%q known=%v auth=%v", tc.code, pe.Prefix(), pe.Subcode(), pe.Normalized(), pe.IsKnown(), pe.IsAuth())
		}
	}
	pe := ProtocolError{Code: CodeAccountFailed}
	if !pe.Is(CodeAccountFailed) || !pe.Is(CodeAccount) || pe.Is(CodeConnection) || pe.Is(CodeAccountMissingData) {
		t.Fatal("Is matching failed")
	}
	if (ProtocolError{Code: "act-auth"}).Prefix() != "act-auth" {
		t.Fatal("hyphenated codes are not protocol codes and must not be split")
	}
}
