// Package simplefin implements the application side of the SimpleFIN
// protocol, a read-only protocol for fetching bank balances and transactions
// from a SimpleFIN server such as the hosted SimpleFIN Bridge.
//
// The typical flow is:
//
//  1. A user visits the server's /create page and receives a setup token.
//  2. The application calls [ClaimSetupToken] to exchange that one-time token
//     for an [AccessURL], which embeds HTTP Basic Auth credentials.
//  3. The application stores the access URL securely and constructs a
//     [Client] with [NewClient].
//  4. The application calls [Client.Accounts] to fetch balances,
//     transactions, and holdings.
//
// This package speaks SimpleFIN protocol version 2 only. Every /accounts
// request sends version=2, and the response models follow the v2 schema
// plus the extra fields returned by the SimpleFIN Bridge (holdings, payee,
// memo, and merchant category code).
//
// Access URLs are secrets. [AccessURL] redacts its password when printed, but
// the Raw field still holds the full credential and must be stored with the
// same care as the financial data it unlocks.
package simplefin
