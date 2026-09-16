package simplefin_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	simplefin "github.com/criswit/simplefin-sdk"
)

func ExampleClaimSetupToken() {
	ctx := context.Background()
	setupToken := "..." // pasted by the user after visiting the server's /create page

	access, err := simplefin.ClaimSetupToken(ctx, setupToken)
	if errors.Is(err, simplefin.ErrClaimTokenRejected) {
		log.Fatal("setup token was already claimed or is invalid; warn the user it may be compromised")
	}
	if err != nil {
		log.Fatal(err)
	}
	// Store access.Raw as a secret. Printing access itself redacts the password.
	fmt.Println(access)
}

func ExampleClient_Accounts() {
	ctx := context.Background()
	client, err := simplefin.NewClient("https://user:pass@bridge.simplefin.org/simplefin")
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now().AddDate(0, 0, -30)
	set, err := client.Accounts(ctx, simplefin.AccountsOptions{StartDate: &start, Pending: true})
	if errors.Is(err, simplefin.ErrAuthenticationFailed) {
		log.Fatal("access URL is invalid or was revoked")
	}
	if err != nil {
		log.Fatal(err)
	}

	for _, pe := range set.ErrList {
		// Always show protocol errors to the user, sanitized for the display medium.
		switch {
		case pe.IsAuth():
			fmt.Println("re-authentication needed:", pe.ConnID)
		case pe.Is(simplefin.CodeAccount):
			fmt.Println("account problem:", pe.AccountID, pe.Normalized())
		}
	}
	for _, acct := range set.Accounts {
		fmt.Println(acct.Name, acct.Balance, acct.Currency)
		for _, tx := range acct.Transactions {
			fmt.Println("  ", tx.EffectiveTime().Format(time.DateOnly), tx.Amount, tx.Payee, tx.MCC)
		}
		for _, h := range acct.Holdings {
			fmt.Println("  ", h.Symbol, h.Shares, h.MarketValue)
		}
	}
}

func ExampleClient_CurrencyInfo() {
	ctx := context.Background()
	client, err := simplefin.NewClient("https://user:pass@bridge.simplefin.org/simplefin")
	if err != nil {
		log.Fatal(err)
	}
	info, err := client.CurrencyInfo(ctx, "https://rewards.example.com/currency/points")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info.Name, info.Abbr)
}
