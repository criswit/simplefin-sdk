//go:build integration

package simplefin_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/criswit/simplefin-sdk"
)

// demoAccessURL is the public SimpleFIN Bridge demo server. It needs no
// account and is used when SIMPLEFIN_ACCESS_URL is unset.
const demoAccessURL = "https://demo:demo@beta-bridge.simplefin.org/simplefin"

func integrationClient(t *testing.T) *simplefin.Client {
	t.Helper()
	accessURL := os.Getenv("SIMPLEFIN_ACCESS_URL")
	if accessURL == "" {
		accessURL = demoAccessURL
	}
	client, err := simplefin.NewClient(accessURL)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestIntegrationInfo(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := client.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Versions) == 0 {
		t.Fatal("expected at least one supported protocol version")
	}
}

func TestIntegrationAccountsBalancesOnly(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accountSet, err := client.Accounts(ctx, simplefin.AccountsOptions{BalancesOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if accountSet == nil {
		t.Fatal("expected account set")
	}
	if len(accountSet.Connections) == 0 || len(accountSet.Accounts) == 0 {
		t.Fatalf("expected v2 connections and accounts, got %d/%d", len(accountSet.Connections), len(accountSet.Accounts))
	}
	for _, pe := range accountSet.ErrList {
		t.Logf("protocol error %s: %s", pe.Code, pe.Message)
	}
}

func TestIntegrationAccountsWithTransactions(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now().AddDate(0, 0, -30)
	accountSet, err := client.Accounts(ctx, simplefin.AccountsOptions{StartDate: &start, Pending: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range accountSet.Accounts {
		if a.ConnID == "" {
			t.Fatalf("account %q has no conn_id; server did not honor version=2", a.ID)
		}
	}
}
