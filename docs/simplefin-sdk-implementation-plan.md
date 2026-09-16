# SimpleFIN SDK Implementation Plan

## Objective

Create a true SimpleFIN SDK as a new Go package in this repository:

```text
simplefin/
```

The package should fully model the SimpleFIN protocol and own all protocol-level behaviour: setup-token claiming, access URL parsing, HTTPS enforcement, Basic Auth, `/info`, `/accounts`, query construction, typed HTTP errors, structured SimpleFIN errors, and complete v2 response models.

It must not own chi-chi-moni application concerns such as AWS Secrets Manager, SQLite persistence, sync jobs, scheduling, or logging policy.

## Reference Links

- Official SimpleFIN Protocol v2 draft: https://www.simplefin.org/protocol.html
- SimpleFIN Bridge Developer Guide: https://beta-bridge.simplefin.org/info/developers
- Protocol source markdown: https://github.com/simplefin/simplefin.github.com/blob/master/protocol.md
- Protocol v1 reference: https://www.simplefin.org/protocol-v1.html
- Bridge create URL: https://bridge.simplefin.org/simplefin/create

## Current Repository Context

Current relevant files:

```text
api/access_token.go       # setup-token/access-token logic, currently manual URL parsing
api/client.go             # SimpleFIN HTTP client, currently incomplete status/error handling
api/roundTripper.go       # Basic Auth transport
model/account.go          # current models, incomplete/outdated for v2
main.go                   # app orchestration; currently calls api + aws + db directly
aws/secrets_manager.go    # currently stores/retrieves api.AccessToken
db/client.go              # app persistence, should remain outside SDK
```

Current tests pass with:

```bash
go test ./...
```

## Package Layout

Create:

```text
simplefin/
  client.go              # Client, constructor, options, shared request/response helpers
  auth.go                # SetupToken, AccessURL, ClaimSetupToken, ParseAccessURL
  accounts.go            # GET /accounts and AccountsOptions
  info.go                # GET /info
  currency.go            # custom currency metadata helper
  models.go              # protocol data models
  errors.go              # HTTPError, sentinel errors, ProtocolError helpers

  auth_test.go
  client_test.go
  accounts_test.go
  info_test.go
  currency_test.go
  models_test.go
  errors_test.go
```

## Public SDK Interface

### Client

```go
type Client struct {
	baseURL    *url.URL
	username   string
	password   string
	httpClient *http.Client
	version    int
}

func NewClient(accessURL string, opts ...Option) (*Client, error)

type Option func(*Client)

func WithHTTPClient(client *http.Client) Option
func WithVersion(version int) Option
```

Usage:

```go
client, err := simplefin.NewClient(accessURL)
if err != nil {
	return err
}

accountSet, err := client.Accounts(ctx, simplefin.AccountsOptions{
	BalancesOnly: true,
})
```

Default protocol version should be `2`.

### Auth / Access URL

```go
type AccessURL struct {
	Raw      string
	BaseURL  string
	Username string
	Password string
}

func ParseAccessURL(raw string) (AccessURL, error)
func ClaimSetupToken(ctx context.Context, setupToken string, opts ...ClaimOption) (AccessURL, error)
```

Implementation requirements:

- Setup Token is a Base64-encoded claim URL.
- Base64-decode it.
- Decoded claim URL must be HTTPS.
- Claim with HTTP `POST` and empty body / zero content length.
- `403` from claim means token does not exist, was already claimed, or may be compromised.
- Successful claim response body is an Access URL.
- Access URL is a URL with embedded Basic Auth credentials:
  ```text
  https://user:pass@bridge.simplefin.org/simplefin
  ```
- Access URL must be HTTPS.
- Access URL must contain username and password.
- Use `net/url`, not manual string splitting.
- Store the credential-free base URL separately from username/password.

### GET /info

```go
type InfoResponse struct {
	Versions []string `json:"versions"`
}

func (c *Client) Info(ctx context.Context) (*InfoResponse, error)
```

Request path:

```text
GET {root}/info
```

### GET /accounts

```go
type AccountsOptions struct {
	StartDate    *time.Time
	EndDate      *time.Time
	Pending      bool
	AccountIDs   []string
	BalancesOnly bool
	Version      int
}

func (c *Client) Accounts(ctx context.Context, opts AccountsOptions) (*AccountSet, error)
```

Request path:

```text
GET {root}/accounts
```

Query params:

- `version=2` by default.
- `start-date=<unix seconds>`: optional, inclusive.
- `end-date=<unix seconds>`: optional, exclusive.
- `pending=1`: optional, include only when true.
- `account=<id>`: optional, repeat once per account ID.
- `balances-only=1`: optional, include only when true.

Bridge-specific note from docs:

- `/accounts` date range is limited to 90 days at a time by the Bridge.
- Do not hard-fail all clients by default on this because it is Bridge-specific; instead add docs and maybe a helper later.

### Custom Currency Info

SimpleFIN supports custom currency URLs. Add:

```go
type CurrencyInfo struct {
	Name string `json:"name"`
	Abbr string `json:"abbr"`
}

func (c *Client) CurrencyInfo(ctx context.Context, currencyURL string) (*CurrencyInfo, error)
```

Rules:

- Currency URL must be HTTPS.
- Fetch and decode JSON object with `name` and `abbr`.
- Do not sanitize strings inside SDK; document that callers must sanitize before display.

## Complete v2 Models

Use SimpleFIN protocol v2 object names and fields.

```go
type AccountSet struct {
	ErrList     []ProtocolError `json:"errlist"`
	Errors      []string        `json:"errors,omitempty"` // deprecated in v2, preserve for compatibility
	Connections []Connection    `json:"connections"`
	Accounts    []Account       `json:"accounts"`
}

type ProtocolError struct {
	Code      string `json:"code"`
	Message   string `json:"msg"`
	ConnID    string `json:"conn_id,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Extra     Extra  `json:"-"`
}

type Connection struct {
	ConnID  string `json:"conn_id"`
	Name    string `json:"name"`
	OrgID   string `json:"org_id"`
	OrgName string `json:"org_name,omitempty"`
	OrgURL  string `json:"org_url,omitempty"`
	SfinURL string `json:"sfin_url"`
	Extra   Extra  `json:"-"`
}

type Account struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	ConnID           string        `json:"conn_id"`
	Currency         string        `json:"currency"`
	Balance          NumericString `json:"balance"`
	AvailableBalance NumericString `json:"available-balance,omitempty"`
	BalanceDate      UnixTime      `json:"balance-date"`
	Transactions     []Transaction `json:"transactions,omitempty"`
	Extra            Extra         `json:"extra,omitempty"`
}

type Transaction struct {
	ID           string        `json:"id"`
	Posted       UnixTime      `json:"posted"`
	Amount       NumericString `json:"amount"`
	Description  string        `json:"description"`
	TransactedAt *UnixTime     `json:"transacted_at,omitempty"`
	Pending      bool          `json:"pending,omitempty"`
	Extra        Extra         `json:"extra,omitempty"`
}

type NumericString string

type UnixTime int64

type Extra map[string]json.RawMessage
```

Important model notes:

- Preserve numeric strings as strings. Do not parse money into `float64`.
- `errlist` is structured and should be first-class.
- `errors` is deprecated but should still decode.
- v2 replaces the old nested Organization with flatter `Connection` objects.
- `Account.conn_id` is required in v2.
- Optional fields include `available-balance`, `transactions`, account `extra`, transaction `transacted_at`, transaction `pending`, transaction `extra`.
- Preserve arbitrary `extra` JSON as `json.RawMessage` values.
- Consider custom `UnmarshalJSON` only if needed to preserve unknown top-level fields; otherwise keep the model simple.

Helpers:

```go
func (t UnixTime) Time() time.Time
func (a Account) BalanceTime() time.Time
func (tx Transaction) PostedTime() time.Time
func (e ProtocolError) Prefix() string // gen, con, act
func (e ProtocolError) IsAuth() bool
func (a AccountSet) HasErrors() bool
func (a AccountSet) ProtocolErrors() ProtocolErrors
```

## Error Design

Create typed HTTP and protocol errors.

```go
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *HTTPError) Error() string
```

```go
type ProtocolErrors []ProtocolError

func (e ProtocolErrors) Error() string
```

Sentinel errors:

```go
var ErrInsecureURL = errors.New("simplefin: insecure URL")
var ErrMissingCredentials = errors.New("simplefin: access URL missing credentials")
var ErrClaimTokenRejected = errors.New("simplefin: setup token rejected or already claimed")
var ErrPaymentRequired = errors.New("simplefin: payment required")
var ErrAuthenticationFailed = errors.New("simplefin: authentication failed")
```

Endpoint behaviour:

- Non-2xx returns `*HTTPError`.
- Claim `403` should be identifiable with `errors.Is(err, ErrClaimTokenRejected)`.
- Accounts `402` should be identifiable with `errors.Is(err, ErrPaymentRequired)`.
- Accounts `403` should be identifiable with `errors.Is(err, ErrAuthenticationFailed)`.
- Successful `/accounts` response may include `errlist`; return response normally. Do not fail the call just because `errlist` is non-empty, because partial data can still be useful.

Implementation suggestion for wrapping:

- Either create custom error types with `Unwrap() error`, or use `fmt.Errorf("%w: %w", sentinel, httpErr)` carefully.
- Ensure tests verify `errors.Is` works.

## HTTP Implementation Requirements

- All request methods accept `context.Context`.
- Use `http.NewRequestWithContext`.
- Use Basic Auth from parsed Access URL.
- Never include credentials in request URL logs/errors if avoidable.
- Enforce HTTPS for:
  - claim URL
  - access URL
  - custom currency URL
- Verify TLS certificates by default; do not disable TLS verification.
- Allow a custom `*http.Client` via `WithHTTPClient` for tests, tracing, retries, and timeouts.
- Do not mutate unexported client internals in tests; use constructor options.

## Testing Plan

### Auth tests

- valid setup token claims access URL
- invalid base64 fails
- decoded non-HTTPS claim URL fails with `ErrInsecureURL`
- claim `403` returns `ErrClaimTokenRejected`
- claim non-2xx returns `HTTPError`
- access URL without credentials fails with `ErrMissingCredentials`
- access URL with missing password fails with `ErrMissingCredentials`
- access URL with escaped username/password parses correctly
- access URL strips credentials from base URL
- HTTP access URL rejected with `ErrInsecureURL`

### Client tests

- default protocol version is `2`
- `WithVersion` overrides version
- `WithHTTPClient` is used
- malformed access URL rejected
- insecure access URL rejected

### Accounts tests

- default request sends `version=2`
- `balances-only=1` included when true
- `pending=1` included when true
- repeated `account=` params for multiple account IDs
- `start-date` encoded as Unix seconds
- `end-date` encoded as Unix seconds
- Basic Auth is sent
- `200` parses complete v2 model
- `402` returns typed payment error
- `403` returns typed authentication error
- malformed JSON returns decode error
- `errlist` is preserved without failing response
- deprecated `errors` is preserved

### Info tests

- `GET /info` path is correct
- Basic Auth is sent or not? Protocol `/info` does not require auth; prefer not sending auth for `/info` if using server root directly. If `Info` is attached to authenticated client, sending auth is acceptable but document behaviour.
- parses supported versions
- non-2xx returns `HTTPError`

### Currency tests

- HTTPS custom currency URL parses
- HTTP custom currency URL rejected with `ErrInsecureURL`
- malformed JSON fails
- non-2xx returns `HTTPError`

### Model tests

- complete AccountSet JSON roundtrip
- optional fields omitted safely
- `extra` fields preserved
- custom currency URL accepted as `Currency`
- numeric strings remain exact strings
- UnixTime helpers work
- ProtocolError prefix/auth helpers work

## Migration Plan for chi-chi-moni

After SDK package is implemented:

1. Add `simplefin/` package without deleting old `api`/`model` yet.
2. Update `main.go` to consume the SDK:
   ```go
   sf, err := simplefin.NewClient(accessURL)
   resp, err := sf.Accounts(ctx, simplefin.AccountsOptions{
       BalancesOnly: true,
   })
   ```
3. Update credential retrieval to return raw Access URL or `simplefin.AccessURL`, not `api.AccessToken`.
4. Update `aws/secrets_manager.go` to avoid importing the old `api` package. Ideally store/retrieve a raw access URL string or a small app-owned secret DTO.
5. Add explicit mapping from `simplefin.Account` to db persistence input.
6. Do not let `db` depend on `simplefin` unless intentionally choosing to persist protocol objects directly. Prefer an app-owned persistence input to keep SDK and persistence separate.
7. Delete or deprecate old `api` package after migration.
8. Delete or shrink old `model` package if SDK models replace it.
9. Run:
   ```bash
   go test ./...
   ```

## First Vertical Slice

Recommended implementation order:

1. `simplefin/models.go`
2. `simplefin/errors.go`
3. `simplefin/auth.go`
4. `simplefin/client.go`
5. `simplefin/accounts.go`
6. `simplefin/info.go`
7. `simplefin/currency.go`
8. Tests for auth + access URL parsing
9. Tests for `/accounts` request construction + response parsing
10. Update chi-chi-moni to use the package

This creates the main SDK seam first:

```go
simplefin.Client.Accounts(ctx, opts)
```

That seam provides high leverage: callers learn one small interface while the implementation owns URL parsing, authentication, query construction, HTTP status semantics, and complete protocol models.

## Acceptance Criteria

- New `simplefin` package exists.
- Package has complete v2 models from official protocol docs.
- SDK can claim setup tokens.
- SDK can create a client from an Access URL.
- SDK rejects insecure URLs.
- SDK performs `GET /accounts?version=2` with Basic Auth.
- SDK exposes `/info`.
- SDK exposes custom currency metadata fetch.
- SDK returns typed HTTP/status errors.
- SDK preserves structured `errlist` and deprecated `errors`.
- SDK tests cover request construction, auth, model decoding, errors, and HTTPS validation.
- `go test ./...` passes.
- chi-chi-moni app can be migrated to use the new package without SDK depending on AWS or SQLite.
