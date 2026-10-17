# AGENTS.md

Instructions for AI agents and contributors working in this repository.

## What this is

`github.com/fatkulnurk/duitku-go` is a Go client for the Duitku Payment Gateway
API v2 (Indonesian payment gateway). It is a single flat package named `duitku`.

Two properties define almost every decision in this codebase:

1. **Standard library only.** There are no third party dependencies and no
   `go.sum`. Do not add one. If you think you need a dependency, you do not.
2. **HMAC-SHA256 signatures.** Every request is signed and every callback is
   verified with HMAC-SHA256. The old MD5 and plain SHA256 schemes are obsolete
   and must never be reintroduced.

## Install

```bash
go get github.com/fatkulnurk/duitku-go
```

Import path is `github.com/fatkulnurk/duitku-go`, package name is `duitku`.

Requires Go 1.27 or newer, because the package uses `encoding/json/v2`.

## Commands

Run these from the repository root before claiming any task is done.

```bash
gofmt -l .              # must print nothing
go build ./...          # must exit 0
go vet ./...            # must exit 0
go test ./...           # must pass
go test -cover ./...    # expect ~99% statement coverage
go test -race ./...     # requires cgo and a C compiler
```

`-race` needs `CGO_ENABLED=1` and gcc. On Windows without gcc it will fail to
build; that is an environment limit, not a code problem. CI runs it on Ubuntu.

## Repository map

| File | Contents |
|---|---|
| `client.go` | `Config`, `Client`, `NewClient`, `doRequest`, `createSignatureHMACSHA256`, `APIError`, base URL constants |
| `transaction.go` | `TransactionRequest` and nested types, `TransactionResponse`, `CreateTransaction`, `CheckTransaction` |
| `payment_methods.go` | `PaymentMethod`, `GetPaymentMethods` |
| `callback.go` | `CallbackData`, `ParseCallback`, `VerifyCallbackSignature`, `HandleCallback` |
| `constants.go` | Payment method codes, status codes, documented limits |
| `expiry.go` | `ExpiryPeriodLimits`, `ExpiryPeriodFor`, `ValidateExpiryPeriod` |
| `validation.go` | `TransactionRequest.Validate` and its helpers |
| `types.go` | `StringOrInt` |
| `doc.go` | Package level documentation (also the `pkg.go.dev` landing page) |
| `example/main.go` | Runnable demo web server |

Tests are table driven and live beside the code: `client_test.go`,
`transaction_test.go`, `callback_test.go`, `payment_methods_test.go`,
`types_test.go`, `expiry_test.go`, `validation_test.go`.

## Public API

### Client lifecycle

```go
func NewClient(config Config) *Client

type Config struct {
    MerchantCode               string
    APIKey                     string
    IsSandbox                  bool
    HTTPClient                 *http.Client  // optional
    Logger                     *log.Logger   // optional
    LogEveryRequestAndResponse bool
}
```

A nil `HTTPClient` gets a 30 second timeout. A supplied client with a zero
timeout is copied and given the same 30 second default. Every API call fails
fast with an error when `MerchantCode` or `APIKey` is empty.

### Methods

```go
func (c *Client) GetPaymentMethods(amount int) ([]PaymentMethod, error)
func (c *Client) CreateTransaction(request TransactionRequest) (*TransactionResponse, error)
func (c *Client) CheckTransaction(merchantOrderID string) (*TransactionStatusResponse, error)
func (c *Client) ParseCallback(r *http.Request) (*CallbackData, error)
func (c *Client) VerifyCallbackSignature(data *CallbackData) bool
func (c *Client) HandleCallback(w http.ResponseWriter, r *http.Request, handler func(*CallbackData) error)
```

### Request types

```go
type TransactionRequest struct {
    // Required
    PaymentAmount   int
    MerchantOrderID string
    ProductDetails  string
    Email           string
    PaymentMethod   string
    CustomerVaName  string
    ReturnURL       string
    CallbackURL     string

    // Optional
    ExpiryPeriod     int
    AdditionalParam  string
    MerchantUserInfo string
    PhoneNumber      string
    ItemDetails      []ItemDetail
    CustomerDetail   *CustomerDetail
    AccountLink      *AccountLink
    CreditCardDetail *CreditCardDetail
}

type ItemDetail struct {
    Name     string
    Price    int
    Quantity int
}

type CustomerDetail struct {
    FirstName       string
    LastName        string
    Email           string
    PhoneNumber     string
    BillingAddress  *Address
    ShippingAddress *Address
}

type Address struct {
    FirstName, LastName, Address, City, PostalCode, Phone, CountryCode string
}

type AccountLink struct {
    CredentialCode string
    OVO            *OVODetail
    Shopee         *ShopeeDetail
}

type OVODetail struct {
    PaymentDetails []OVOPaymentDetail
}

type OVOPaymentDetail struct {
    PaymentType string // "CASH"
    Amount      int
}

type ShopeeDetail struct {
    UseCoin bool
    PromoID string // json key "promoId"
}

type CreditCardDetail struct {
    Acquirer     string   // "014" BCA, "022" CIMB
    BinWhitelist []string // 3 digit bank code or 6 digit card BIN
}
```

### Response types

```go
type TransactionResponse struct {
    MerchantCode  string
    Reference     string
    PaymentURL    string
    VANumber      string
    QrString      string
    AppURL        string
    Amount        StringOrInt
    StatusCode    string
    StatusMessage string
}

type TransactionStatusResponse struct {
    MerchantOrderID string
    Reference       string
    Amount          StringOrInt
    Fee             string
    StatusCode      string
    StatusMessage   string
}

type PaymentMethod struct {
    PaymentMethod string
    PaymentName   string
    PaymentImage  string
    TotalFee      string
}

type CallbackData struct {
    MerchantCode     string
    Amount           string
    MerchantOrderID  string
    ProductDetail    string
    AdditionalParam  string
    PaymentCode      string
    ResultCode       string
    MerchantUserID   string
    Reference        string
    Signature        string
    PublisherOrderId string
    SpUserHash       string
    SettlementDate   string
    IssuerCode       string
    CustomerName     string
}

func (data *CallbackData) IsSuccessful() bool
func (data *CallbackData) IsFailed() bool
```

### Validation and expiry helpers

```go
func (r TransactionRequest) Validate() error
func ExpiryPeriodFor(paymentMethod string) (ExpiryPeriodLimits, bool)
func ValidateExpiryPeriod(paymentMethod string, expiryPeriod int) error

type ExpiryPeriodLimits struct {
    Default int  // minutes Duitku uses when expiryPeriod is not sent
    Max     int  // 0 means the docs publish no maximum
    Fixed   bool // Duitku ignores the requested value entirely
}
```

`Validate` is opt in. `CreateTransaction` does not call it, because Duitku is
the source of truth. Call it when you want to fail fast with a clear local
error instead of an opaque HTTP 400.

### Errors

```go
type APIError struct {
    StatusCode int    // HTTP status from Duitku
    Code       string // Duitku code when present
    Message    string // reason from Duitku
    RawBody    string // raw body, truncated to 256 runes
}
```

Non-200 responses return `*APIError`. Use `errors.As` to inspect it. Body
parsing is best effort: an HTML gateway page, an empty body, or a rate limit
block is reported as an HTTP failure, never as a JSON decoding error.

## JSON encoding

The package uses `encoding/json/v2` (`encoding/json/jsontext` for tokens).
In Go 1.27 the v2 experiment is part of the toolchain baseline, so no
`GOEXPERIMENT` flag is needed, and `encoding/json` is itself implemented on top
of v2.

`duitkuJSONOptions` in `client.go` is applied to every payload and sets two
options that are deliberately more lenient than the v2 defaults:

- `MatchCaseInsensitiveNames(true)`, because Duitku's docs disagree about key
  casing (`AppUrl` in one place, `appUrl` in another).
- `AllowDuplicateNames(true)`, because v2 rejects duplicate object members
  while v1 permitted them, and a repeated key must not fail a payment response.

Do not "modernize" these away. Removing either one makes the client reject
valid Duitku responses. `TestJSONOptionsTolerance` in `transaction_test.go`
pins both.

Two v2 differences are easy to get wrong when editing struct tags:

- `omitempty` under v2 omits only JSON-empty values, so a zero `int` or `false`
  is **kept**. Use `omitzero` for Go zero values. `ExpiryPeriod` relies on this;
  with `omitempty` the client would send `"expiryPeriod":0`.
- Unmarshaling is case sensitive by default, unlike v1.

`StringOrInt` implements the v2 `MarshalerTo` and `UnmarshalerFrom` interfaces.
Do not add `MarshalJSON`/`UnmarshalJSON` back: `encoding/json` already dispatches
to the v2 methods, so they would be dead code.

## Signature specification

Every operation uses `HMAC_SHA256(stringToSign, apiKey)` encoded as lowercase
hex. The API key is the HMAC key. It is not appended to the message.

| Operation | stringToSign |
|---|---|
| Get Payment Method | `merchantcode + amount + datetime` |
| Create Transaction | `merchantCode + merchantOrderId + paymentAmount` |
| Check Transaction | `merchantCode + merchantOrderId` |
| Callback | `merchantCode + amount + merchantOrderId` |

Fixed test vectors, using API key `DXXXXCX80TZJ85Q70QCI` and merchant code
`DXXXX`:

| Operation | stringToSign | Expected signature |
|---|---|---|
| Get Payment Method | `DXXXX100002022-01-25 16:23:08` | `211cd5de6edf73a597ee6847e63d342346fbd0212604e6cbe544d97788817c82` |
| Create Transaction | `DXXXXORDER12340000` | `04c5dfd9467fd81ab3ee6def7e46d2af044bcf64208b6db998256d103e1b73ad` |
| Check Transaction | `DXXXXORDER123` | `b80fd7de2ea8d95da86994ea0ba547bcdcd4451935929ee85ce52f04f3d98fd0` |
| Callback | `DXXXX40000ORDER123` | `48aed82d03914dc87da05226fd6c60b89f9a28ba81089e347e4c1de6552e2bfd` |

A correct signature is always 64 lowercase hex characters. If you see 32, you
have produced an MD5 digest and something is wrong.

## Invariants

Breaking any of these is a bug:

1. **Use `hmac.New(sha256.New, []byte(apiKey))`.** Never `sha256.Sum256` over a
   string that has the API key concatenated onto it. Those are different
   functions and produce different output.
2. **Verify signatures with `hmac.Equal`.** Never `==` and never
   `strings.EqualFold`. The comparison must be constant time and case
   sensitive, because the docs mandate lowercase hex.
3. **No third party imports.** `go.mod` has no `require` block.
4. **All tests are table driven.** Every `TestXxx` function declares a slice
   named `tests` with named cases and iterates it with
   `for _, tt := range tests`. No ad hoc assertions outside that shape.
5. **Signatures are asserted against fixed literals.** Do not compute the
   expected value with the same helper the production code uses; that is a
   circular test that passes even when the algorithm is wrong.
6. **Tests never touch the real network.** Use `httptest.NewServer` or an
   injected `http.RoundTripper`.
7. **Every JSON payload goes through `duitkuJSONOptions`.** Never call
   `json.Marshal` or `json.Unmarshal` without them. Never use `omitempty` on a
   Go zero value where omission matters; use `omitzero`. See "JSON encoding".

## Gotchas

These are real behaviours that trip people up. They are all intentional.

- **`merchantcode` is lowercase on exactly one endpoint.** Get Payment Method
  sends `merchantcode`. Everything else sends `merchantCode`. This is Duitku's
  inconsistency, documented in `payment_methods.go`.
- **`amount` is a string in some responses and a number in others.** The docs
  tables say integer, the docs examples quote it. `StringOrInt` decodes both.
  Call `.String()` or `.Int()` rather than comparing to a literal.
- **`AppUrl` casing is inconsistent in the docs.** The JSON example uses
  `AppUrl`, the parameter table uses `appUrl`. `duitkuJSONOptions` enables
  case-insensitive name matching, so both decode into
  `TransactionResponse.AppURL`.
- **Callback `resultCode` is not covered by the signature.** A captured
  callback can have its result flipped without breaking the HMAC. Always
  confirm with `CheckTransaction` before releasing goods or credit.
- **Callback `01` and redirect `01` mean different things.** On a callback, `01`
  is Failed. On a redirect, `01` is Pending and `02` is Canceled. The constants
  are kept in separate groups on purpose. Never use the redirect `resultCode`
  to change a payment status, because `returnUrl` is user controlled.
- **`HandleCallback` answers 500 when the handler returns an error.** That is
  deliberate: Duitku retries up to five times and a transient failure should be
  retried. The handler error is logged, never echoed to the caller.
- **`Validate` treats `expiryPeriod` 0 as "use the channel default"** and
  rejects a value on channels where Duitku ignores it entirely (Credit Card,
  OVO Account Link, Shopee Account Link).
- **The docs' own inquiry example breaks the item total rule.**
  `10000*1 + 30000*3 = 100000` against a `paymentAmount` of `40000`. Do not
  copy that example into a test as valid input.
- **`VoidTransaction` and the subscription fields do not exist.** They were
  removed because Duitku does not document them in API v2. Do not add them
  back.

## Adding a payment method

1. Add the constant to the right block in `constants.go`. Keep Duitku's
   category from the payment method table.
2. If the docs publish an expiry window for it, add an entry to
   `expiryPeriodLimits` in `expiry.go`.
3. If the channel has extra parameter requirements, extend
   `validateChannelRequirements` in `validation.go`.
4. Add it to the table in `doc.go`, the table in `README.md`, and the dropdown
   in `example/main.go`.
5. Add a case to the relevant tables in `expiry_test.go` and
   `validation_test.go`.
6. Run the full command list above.

## Common tasks

**Verify a signature is correct.** Compare against the fixed vectors in the
table above. Do not trust a test that recomputes the expected value with the
same helper.

**Add a new API endpoint.** Add the method to the matching file, route it
through `c.doRequest`, and add a table driven test that asserts the exact URL
path, the exact `Content-Type`, the signature, and the decoded response.

**Change a request or response struct.** Update the struct, then update
`transaction_test.go` or `payment_methods_test.go`. Those tests decode the raw
request body with explicit json tags, so a tag regression fails the build.

**Debug a failing integration.** Turn on `LogEveryRequestAndResponse` in
`Config`. The logger prints the URL, the body, the status and the raw response.
