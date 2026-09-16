# simplefin-sdk

Go SDK for the [SimpleFIN](https://www.simplefin.org/protocol.html) protocol.

```bash
go get github.com/criswit/simplefin-sdk
```

```go
import simplefin "github.com/criswit/simplefin-sdk"
```

## What SimpleFIN is

SimpleFIN is a read-only financial-data protocol. A user authorizes an application to read balances and transactions from a SimpleFIN Server, which may be operated directly by a financial institution or by the hosted SimpleFIN Bridge.

The protocol has four server endpoints:

- `GET /create` - user-facing flow that creates a setup token.
- `POST /claim/:token` - exchanges a setup token for an access URL.
- `GET /info` - reports supported protocol versions.
- `GET /accounts` - returns balances, transactions, connections, and protocol-level errors.

This SDK implements the application side of the protocol: setup-token claiming, access URL parsing, HTTPS enforcement, Basic Auth, `/info`, `/accounts`, custom currency metadata lookup, typed HTTP errors, and v2 response models.

```mermaid
flowchart LR
    User[User] -->|visits /create| Server[SimpleFIN Server or Bridge]
    Server -->|setup token| User
    User -->|pastes setup token| App[Your Go app]
    App -->|ClaimSetupToken: POST decoded claim URL| Server
    Server -->|access URL with Basic Auth credentials| App
    App -->|NewClient + GET /accounts| Server
    Server -->|AccountSet JSON| App

    subgraph SDK[github.com/criswit/simplefin-sdk]
        Claim[ClaimSetupToken]
        Parse[ParseAccessURL]
        Client[Client]
        Models[Typed response models]
    end

    App --> Claim
    App --> Parse
    App --> Client
    Client --> Models
```

## Authentication model

SimpleFIN authentication has two separate credentials/URLs that are easy to confuse:

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant App as Your application
    participant SFIN as SimpleFIN Server / Bridge

    User->>SFIN: Open /create in browser
    SFIN-->>User: Copyable setup token<br/>(Base64 claim URL)
    User->>App: Paste setup token
    App->>App: Base64 decode setup token
    App->>SFIN: POST decoded HTTPS claim URL
    alt token valid and unused
        SFIN-->>App: 200 access URL<br/>https://user:pass@host/simplefin
        App->>App: Store access URL securely
        App->>SFIN: GET /accounts with Basic Auth
        SFIN-->>App: 200 AccountSet JSON
    else token rejected or already used
        SFIN-->>App: 403 Forbidden
        App->>User: Warn token may be compromised or expired
    end
```

### 1. Setup token / SimpleFIN token

A setup token is what the user gives your app after completing the server's `/create` flow. The hosted Bridge create URL is:

```text
https://bridge.simplefin.org/simplefin/create
```

The setup token is not directly used for `/accounts`. It is a Base64-encoded claim URL. For example, decoding a setup token yields something like:

```text
https://bridge.simplefin.org/simplefin/claim/ONE_TIME_CODE
```

Setup tokens are one-time use in normal operation. After your app claims one, store the returned access URL and do not expect the setup token to work again.

### 2. Claiming an access URL

To claim a setup token, decode it and `POST` to the decoded HTTPS claim URL with an empty body:

```go
accessURL, err := simplefin.ClaimSetupToken(ctx, setupToken)
if err != nil {
    // Handle invalid Base64, insecure HTTP URLs, network errors, and rejected tokens.
}

// Persist accessURL.Raw securely for future API calls.
```

On success, the server returns an access URL in the response body:

```text
https://username:password@bridge.simplefin.org/simplefin
```

That URL contains Basic Auth credentials. Treat it like a secret and store it at least as securely as the financial data you fetch with it.

On `403 Forbidden` while claiming, the SDK returns an error that matches `simplefin.ErrClaimTokenRejected` and also wraps `*simplefin.HTTPError`:

```go
accessURL, err := simplefin.ClaimSetupToken(ctx, setupToken)
if errors.Is(err, simplefin.ErrClaimTokenRejected) {
    // The setup token does not exist or was already claimed.
    // The protocol recommends warning the user that the token may be compromised.
}
```

### 3. Access URL parsing and Basic Auth

`simplefin.NewClient` accepts the access URL, validates it, removes credentials from the base URL internally, and uses the embedded username/password for HTTP Basic Authentication:

```go
client, err := simplefin.NewClient(accessURL.Raw)
if err != nil {
    return err
}
```

You can also parse an access URL directly:

```go
parsed, err := simplefin.ParseAccessURL("https://user:pass@example.com/simplefin")
// parsed.BaseURL  == "https://example.com/simplefin"
// parsed.Username == "user"
// parsed.Password == "pass"
```

Security behavior:

- Claim URLs and access URLs must be `https://`; `http://` is rejected with `simplefin.ErrInsecureURL`.
- Access URLs must include a non-empty username and password; otherwise `simplefin.ErrMissingCredentials` is returned.
- Client requests set Basic Auth only from the access URL credentials.
- Use the default Go TLS verification behavior unless you have a very specific testing reason not to.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/criswit/simplefin-sdk"
)

func main() {
    ctx := context.Background()

    // Usually obtained once from the user after they visit /create.
    setupToken := "..."

    access, err := simplefin.ClaimSetupToken(ctx, setupToken)
    if err != nil {
        log.Fatal(err)
    }

    // Store access.Raw securely and reuse it later.
    client, err := simplefin.NewClient(access.Raw)
    if err != nil {
        log.Fatal(err)
    }

    start := time.Now().AddDate(0, 0, -30)
    accounts, err := client.Accounts(ctx, simplefin.AccountsOptions{
        StartDate: &start,
        Pending:   true,
        Version:   2,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, acct := range accounts.Accounts {
        fmt.Printf("%s %s %s\n", acct.Name, acct.Balance, acct.Currency)
    }
}
```

## Client configuration

```go
client, err := simplefin.NewClient(
    accessURL,
    simplefin.WithVersion(2),
    simplefin.WithHTTPClient(customHTTPClient),
)
```

Options:

- `WithVersion(version int)` sets the default `version` query parameter for `/accounts`. The SDK defaults to protocol version `2`.
- `WithHTTPClient(client *http.Client)` injects a custom HTTP client for timeouts, transports, tracing, tests, or proxies.

Setup-token claiming has a separate HTTP client option:

```go
access, err := simplefin.ClaimSetupToken(
    ctx,
    setupToken,
    simplefin.WithClaimHTTPClient(customHTTPClient),
)
```

## `/info`

`Info` fetches protocol metadata from `{accessURL}/info`:

```go
info, err := client.Info(ctx)
if err != nil {
    return err
}
fmt.Println(info.Versions) // e.g. []string{"1", "2"}
```

SimpleFIN uses `/info` to advertise supported protocol versions. The current SDK models:

```go
type InfoResponse struct {
    Versions []string `json:"versions"`
}
```

## `/accounts`

`Accounts` fetches account, balance, transaction, connection, and protocol-error data from `{accessURL}/accounts`:

```go
accountSet, err := client.Accounts(ctx, simplefin.AccountsOptions{
    StartDate:    &start,
    EndDate:      &end,
    Pending:      true,
    AccountIDs:   []string{"account-1", "account-2"},
    BalancesOnly: false,
})
```

The SDK turns `AccountsOptions` into query parameters, sends Basic Auth from the access URL, and decodes the v2 `AccountSet` response:

```mermaid
flowchart TD
    A[client.Accounts(ctx, opts)] --> B[Build GET /accounts request]
    B --> C[Add version query<br/>default: 2]
    B --> D[Add optional filters<br/>start-date, end-date, pending,<br/>account, balances-only]
    B --> E[Set HTTP Basic Auth<br/>from access URL credentials]
    C --> F[Send HTTPS request]
    D --> F
    E --> F
    F --> G{HTTP status}
    G -->|200| H[Decode AccountSet]
    G -->|402| I[ErrPaymentRequired + HTTPError]
    G -->|403| J[ErrAuthenticationFailed + HTTPError]
    G -->|other non-2xx| K[HTTPError]
    H --> L[Check AccountSet.ErrList<br/>for protocol-level issues]
```

Supported query options:

| SDK field | Query parameter | Meaning |
| --- | --- | --- |
| `StartDate` | `start-date` | Include transactions posted on or after this Unix timestamp. |
| `EndDate` | `end-date` | Include transactions before, but not on, this Unix timestamp. |
| `Pending` | `pending=1` | Ask the server to include pending transactions if supported. |
| `AccountIDs` | repeated `account` | Restrict results to one or more account IDs. |
| `BalancesOnly` | `balances-only=1` | Skip transaction data and return balances only. |
| `Version` | `version` | Override the client's default protocol version for this request. |

Bridge-specific operational notes:

- The Bridge is designed for daily transaction updates, not high-frequency polling.
- The Bridge developer guide asks apps to stay around 24 requests or fewer per day, with some setup leeway.
- Bridge `/accounts` date ranges are limited to 90 days at a time.
- For scheduled syncs, choose a random minute rather than polling exactly on the hour.
- Overlap transaction windows by about 5 days so late-posting transactions are not missed.

The SDK constructs the query string but does not enforce Bridge-specific limits because non-Bridge SimpleFIN servers may have different policies.

## Response models

### AccountSet

```go
type AccountSet struct {
    ErrList     []ProtocolError `json:"errlist"`
    Errors      []string        `json:"errors,omitempty"`
    Connections []Connection    `json:"connections"`
    Accounts    []Account       `json:"accounts"`
}
```

In protocol v2, `errlist` is the structured error list. The older `errors` string list is deprecated but preserved by the SDK for compatibility.

```mermaid
erDiagram
    AccountSet ||--o{ Connection : contains
    AccountSet ||--o{ Account : contains
    AccountSet ||--o{ ProtocolError : reports
    Connection ||--o{ Account : "conn_id links"
    Account ||--o{ Transaction : contains

    AccountSet {
        ProtocolError[] errlist
        string[] errors "deprecated"
        Connection[] connections
        Account[] accounts
    }
    Connection {
        string conn_id
        string name
        string org_id
        string org_name
        string org_url
        string sfin_url
    }
    Account {
        string id
        string name
        string conn_id
        string currency
        NumericString balance
        NumericString available_balance
        UnixTime balance_date
    }
    Transaction {
        string id
        UnixTime posted
        NumericString amount
        string description
        UnixTime transacted_at
        bool pending
    }
    ProtocolError {
        string code
        string msg
        string conn_id
        string account_id
    }
```

Use helpers:

```go
if accountSet.HasErrors() {
    for _, pe := range accountSet.ProtocolErrors() {
        // Show sanitized user-safe messages where appropriate.
        fmt.Println(pe.Code, pe.Message)
    }
}
```

### Protocol errors

SimpleFIN protocol errors appear inside successful `/accounts` JSON responses. They are different from HTTP errors. Codes use prefixes:

- `gen.*` - general/server/API errors.
- `con.*` - connection-level errors, often tied to a bank login.
- `act.*` - account-level errors.

Known examples include `gen.api`, `gen.auth`, `con.auth`, `act.failed`, and `act.missingdata`. Consumers should handle unknown subcodes by falling back to the prefix.

Helpers:

```go
pe.Prefix() // "gen", "con", or "act" for dotted/hyphenated code styles
pe.IsAuth() // heuristic for auth/login/credential-related errors
```

Always sanitize protocol error messages before displaying them in HTML, terminals, logs, or notifications.

### Account, transactions, and numeric strings

Amounts and balances are modeled as `NumericString`, not `float64`, so exact protocol values are preserved:

```go
type Account struct {
    ID               string
    Name             string
    ConnID           string
    Currency         string
    Balance          NumericString
    AvailableBalance NumericString
    BalanceDate      UnixTime
    Transactions     []Transaction
    Extra            Extra
}
```

`UnixTime` values can be converted to `time.Time`:

```go
balanceTime := account.BalanceTime()
postedTime := transaction.PostedTime()
```

## Custom currencies

A SimpleFIN account currency is usually an ISO 4217 code like `USD`, but it may also be a custom currency URL for points, miles, gift cards, or similar balances. The protocol says that `GET`ing that URL returns:

```json
{
  "name": "Example Airline Miles",
  "abbr": "miles"
}
```

Use:

```go
info, err := client.CurrencyInfo(ctx, account.Currency)
```

`CurrencyInfo` requires HTTPS and returns `simplefin.ErrInsecureURL` for insecure custom currency URLs. Sanitize returned strings before display.

## Error handling

HTTP-level non-2xx responses return `*simplefin.HTTPError`:

```go
var httpErr *simplefin.HTTPError
if errors.As(err, &httpErr) {
    fmt.Println(httpErr.StatusCode, httpErr.Status, string(httpErr.Body))
}
```

Error handling has two layers: HTTP transport/status failures returned as Go errors, and SimpleFIN protocol errors embedded in successful `/accounts` responses.

```mermaid
flowchart TD
    A[SDK call returns] --> B{Go error?}
    B -->|yes| C{errors.Is sentinel?}
    C -->|ErrClaimTokenRejected| D[Setup token invalid/already claimed]
    C -->|ErrAuthenticationFailed| E[Access URL revoked or credentials invalid]
    C -->|ErrPaymentRequired| F[Bridge/server payment required]
    C -->|ErrInsecureURL or ErrMissingCredentials| G[Configuration/security problem]
    C -->|no sentinel| H[Network, JSON, or generic HTTP error]
    B -->|no| I{AccountSet.HasErrors?}
    I -->|yes| J[Read errlist/errors<br/>sanitize before display]
    I -->|no| K[Use accounts and transactions]
```

Common sentinel errors:

| Error | When it occurs |
| --- | --- |
| `ErrInsecureURL` | A claim, access, or custom currency URL is not HTTPS. |
| `ErrMissingCredentials` | An access URL has no username/password. |
| `ErrClaimTokenRejected` | Claim endpoint returns `403`. |
| `ErrPaymentRequired` | `/accounts` returns `402`. |
| `ErrAuthenticationFailed` | `/accounts` or `/info` returns `403`. |

Status sentinels wrap `*HTTPError`, so both checks work:

```go
if errors.Is(err, simplefin.ErrAuthenticationFailed) {
    // Credentials may be wrong, expired, or revoked.
}

var httpErr *simplefin.HTTPError
if errors.As(err, &httpErr) {
    // Inspect status and body.
}
```

## Helper scripts

The `scripts/` directory contains small Bash utilities for manual testing and auth workflows. They intentionally use environment variables or positional arguments so they can be composed in a shell without adding a CLI dependency to the SDK.

| Script | Purpose |
| --- | --- |
| `scripts/claim-setup-token.sh` | Base64-decodes a setup token, verifies the claim URL is HTTPS, POSTs to it, and prints the access URL. |
| `scripts/inspect-access-url.sh` | Parses an access URL and prints redacted components plus derived `/info` and `/accounts` endpoints. |
| `scripts/fetch-info.sh` | Fetches `{ACCESS_URL}/info`. |
| `scripts/fetch-accounts.sh` | Fetches `{ACCESS_URL}/accounts` with optional query parameters. |

Examples:

```bash
# Claim a one-time setup token.
ACCESS_URL=$(scripts/claim-setup-token.sh "$SIMPLEFIN_SETUP_TOKEN")

# Inspect without printing the password.
scripts/inspect-access-url.sh "$ACCESS_URL"

# Fetch server protocol versions.
scripts/fetch-info.sh "$ACCESS_URL"

# Fetch balances only.
BALANCES_ONLY=1 scripts/fetch-accounts.sh "$ACCESS_URL"

# Fetch a filtered transaction window.
START_DATE=1704067200 \
END_DATE=1706745600 \
PENDING=1 \
ACCOUNT_IDS='account-1,account-2' \
scripts/fetch-accounts.sh "$ACCESS_URL"
```

All scripts reject non-HTTPS URLs. Avoid pasting access URLs into shared terminals, logs, CI output, or shell history; access URLs contain Basic Auth credentials.

## Integration testing

Integration tests are guarded by the `integration` build tag and require an access URL:

```bash
SIMPLEFIN_ACCESS_URL='https://user:pass@bridge.simplefin.org/simplefin' \
  go test -tags=integration ./...
```

Regular unit tests use local `httptest` servers:

```bash
go test ./...
```

## Protocol references

- SimpleFIN Protocol v2 draft: <https://www.simplefin.org/protocol.html>
- SimpleFIN Bridge Developer Guide: <https://beta-bridge.simplefin.org/info/developers>
- SimpleFIN protocol source: <https://github.com/simplefin/simplefin.github.com/blob/master/protocol.md>
- SimpleFIN Protocol v1: <https://www.simplefin.org/protocol-v1.html>
