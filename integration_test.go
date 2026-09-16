//go:build integration

package simplefin_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/criswit/simplefin-sdk"
)

func integrationClient(t *testing.T) *simplefin.Client {
	t.Helper()
	accessURL := os.Getenv("SIMPLEFIN_ACCESS_URL")
	if accessURL == "" {
		t.Skip("SIMPLEFIN_ACCESS_URL not set")
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
}
