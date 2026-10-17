---
name: duitku-go
description: Use when writing, reviewing, or debugging Go code that integrates the Duitku payment gateway, or when working inside the github.com/fatkulnurk/duitku-go repository. Covers the HMAC-SHA256 signature scheme, the client API surface, payment method codes, callback verification, request validation, and the documented Duitku quirks (lowercase merchantcode, string-or-int amounts, differing resultCode meanings).
---

# Duitku Go SDK

Integrate the Duitku Payment Gateway API v2 from Go using
`github.com/fatkulnurk/duitku-go`. Standard library only, no dependencies.

## Install

```bash
go get github.com/fatkulnurk/duitku-go
```

```go
import "github.com/fatkulnurk/duitku-go"
```

Package name is `duitku`.

## Quick start

```go
client := duitku.NewClient(duitku.Config{
    MerchantCode: os.Getenv("DUITKU_MERCHANT_CODE"),
    APIKey:       os.Getenv("DUITKU_API_KEY"),
    IsSandbox:    true, // false in production
})

methods, err := client.GetPaymentMethods(10000)

tx := duitku.TransactionRequest{
    PaymentAmount:   40000,
    PaymentMethod:   duitku.PaymentMethodBCA,
    MerchantOrderID: "ORDER123",
    ProductDetails:  "Test Product",
    CustomerVaName:  "John Doe",
    Email:           "customer@example.com",
    CallbackURL:     "https://example.com/callback",
    ReturnURL:       "https://example.com/return",
}
if err := tx.Validate(); err != nil {
    log.Fatal(err) // optional fail-fast check
}

result, err := client.CreateTransaction(tx)
// redirect the customer to result.PaymentURL
```

## The one thing that matters most

Every request is signed with **HMAC-SHA256** where the API key is the HMAC key:

```go
mac := hmac.New(sha256.New, []byte(apiKey))
mac.Write([]byte(stringToSign))
signature := hex.EncodeToString(mac.Sum(nil))
```

The stringToSign differs per operation:

| Operation | stringToSign |
|---|---|
| Get Payment Method | `merchantcode + amount + datetime` |
| Create Transaction | `merchantCode + merchantOrderId + paymentAmount` |
| Check Transaction | `merchantCode + merchantOrderId` |
| Callback | `merchantCode + amount + merchantOrderId` |

**Never** use `md5.Sum`, `sha256.Sum256`, or `h := hash(msg + apiKey)`. Those
are the obsolete schemes and Duitku rejects them. A correct signature is 64
lowercase hex characters; 32 means you produced MD5.

When you use this SDK you never build a signature yourself. The client does it.

## API surface

```go
func NewClient(config Config) *Client

// API calls
func (c *Client) GetPaymentMethods(amount int) ([]PaymentMethod, error)
func (c *Client) CreateTransaction(request TransactionRequest) (*TransactionResponse, error)
func (c *Client) CheckTransaction(merchantOrderID string) (*TransactionStatusResponse, error)

// Callback handling
func (c *Client) HandleCallback(w http.ResponseWriter, r *http.Request, handler func(*CallbackData) error)
func (c *Client) ParseCallback(r *http.Request) (*CallbackData, error)
func (c *Client) VerifyCallbackSignature(data *CallbackData) bool

// Helpers
func (r TransactionRequest) Validate() error
func ExpiryPeriodFor(paymentMethod string) (ExpiryPeriodLimits, bool)
func ValidateExpiryPeriod(paymentMethod string, expiryPeriod int) error
```

## Handling callbacks correctly

```go
http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
    client.HandleCallback(w, r, func(data *duitku.CallbackData) error {
        // Signature already verified, merchant code already matched.
        switch {
        case data.IsSuccessful():
            // Safe to mark paid ONLY after confirming with CheckTransaction,
            // because resultCode is not part of the signed string.
            return markOrderPaid(data.MerchantOrderID, data.Reference)
        case data.IsFailed():
            return markOrderFailed(data.MerchantOrderID)
        }
        return nil
    })
})
```

`HandleCallback` answers HTTP 200 on success and 500 when the handler returns an
error. The 500 is deliberate: Duitku retries up to five times, so a transient
database failure should be retried.

## Checking status correctly

```go
status, err := client.CheckTransaction("ORDER123")
if err != nil {
    return err
}
switch status.StatusCode {
case duitku.CheckTransactionStatusSuccess:   // "00" paid
case duitku.CheckTransactionStatusPending:   // "01" waiting
case duitku.CheckTransactionStatusCancelled: // "02" canceled
}
```

A nil error only means the call succeeded. Never treat nil error as "paid".
Also: do not poll this endpoint aggressively. Duitku throttles it and can block
the caller for about an hour.

## Payment method codes

Always use the constants, never raw strings.

| Category | Constant | Code |
|---|---|---|
| Credit Card | `PaymentMethodCreditCard` | VC |
| Virtual Account | `PaymentMethodBCA` | BC |
| | `PaymentMethodMandiri` | M2 |
| | `PaymentMethodMaybank` | VA |
| | `PaymentMethodBNI` | I1 |
| | `PaymentMethodCIMB` | B1 |
| | `PaymentMethodPermata` | BT |
| | `PaymentMethodATMBersama` | A1 |
| | `PaymentMethodArthaGraha` | AG |
| | `PaymentMethodNeoCommerce` | NC |
| | `PaymentMethodBRI` | BR |
| | `PaymentMethodSahabat` | S1 |
| | `PaymentMethodDanamon` | DM |
| | `PaymentMethodBSI` | BV |
| Retail | `PaymentMethodAlfamart` / `Pegadaian` / `POS` | FT |
| | `PaymentMethodIndomaret` | IR |
| E-Wallet | `PaymentMethodOVO` | OV |
| | `PaymentMethodShopeePay` | SA |
| | `PaymentMethodLinkAjaFixed` | LF |
| | `PaymentMethodLinkAjaPercent` | LA |
| | `PaymentMethodDANA` | DA |
| | `PaymentMethodShopeeLink` | SL |
| | `PaymentMethodOVOLink` | OL |
| E-Banking | `PaymentMethodJeniusPay` | JP |
| QRIS | `PaymentMethodQrisShopeePay` | SP |
| | `PaymentMethodQrisNobu` | NQ |
| | `PaymentMethodQrisNusapay` | SQ |
| Paylater | `PaymentMethodIndodanaPaylater` | DN |
| | `PaymentMethodAtome` | AT |
| E-Commerce | `PaymentMethodTokopediaCard` | T1 |
| | `PaymentMethodTokopediaEWallet` | T2 |
| | `PaymentMethodTokopediaOthers` | T3 |

## Traps to avoid

1. **`merchantcode` is lowercase on Get Payment Method only.** Every other
   endpoint uses `merchantCode`. Do not "fix" this, it is Duitku's spec.
2. **`amount` is sometimes a string, sometimes a number.** Use
   `response.Amount.String()` or `.Int()`. Do not compare to a string literal.
3. **Callback `resultCode` is unsigned.** It is not part of the HMAC, so a
   captured callback can have it flipped. Confirm with `CheckTransaction`.
4. **Callback `01` means Failed. Redirect `01` means Pending, `02` Canceled.**
   Use the matching constant group. Never update payment status from a redirect,
   because `returnUrl` is user controlled.
5. **`expiryPeriod` 0 means "use the channel default".** Do not send 0. Some
   channels (Credit Card, OVO Account Link, Shopee Account Link) ignore the
   requested value entirely and `Validate` rejects a value for them.
6. **Item totals must equal `paymentAmount`.** `sum(price * quantity)` must
   match exactly. The docs' own example violates this, so do not copy it.
7. **Paylater channels require `customerDetail` and `itemDetails`.** E-commerce
   requires `customerVaName`.
8. **Minimum payment is 10000.**
9. **`VoidTransaction` and subscription do not exist** in the documented API v2.

## Error handling

Non-200 responses return `*APIError`:

```go
var apiErr *duitku.APIError
if errors.As(err, &apiErr) {
    log.Printf("Duitku HTTP %d code %s: %s", apiErr.StatusCode, apiErr.Code, apiErr.Message)
}
```

Common documented failures: `Wrong signature` (401), `Merchant not found` and
`Payment channel not available` (404), `Minimum Payment 10000 IDR`,
`Invalid Email Address` (400), `Payment amount must be equal to all item price`
(409).

## Testing integrations

- Signatures can be verified offline against fixed vectors. With API key
  `DXXXXCX80TZJ85Q70QCI` and merchant code `DXXXX`:
  - callback `HMAC("DXXXX40000ORDER123")` =
    `48aed82d03914dc87da05226fd6c60b89f9a28ba81089e347e4c1de6552e2bfd`
  - inquiry `HMAC("DXXXXORDER12340000")` =
    `04c5dfd9467fd81ab3ee6def7e46d2af044bcf64208b6db998256d103e1b73ad`
  - check `HMAC("DXXXXORDER123")` =
    `b80fd7de2ea8d95da86994ea0ba547bcdcd4451935929ee85ce52f04f3d98fd0`
- The client's base URL is unexported, so from outside the package you cannot
  point it at an `httptest.NewServer` directly. Inject a custom `HTTPClient`
  whose `Transport` rewrites the request host to the test server. Inside the
  package, tests build a `Client` literal with `baseURL` set to `server.URL`.
- Sandbox credentials and dummy cards are listed at
  https://docs.duitku.com/api/id/#uji-coba

## Reference

- API docs (Indonesian): https://docs.duitku.com/api/id/
- Get payment method: https://docs.duitku.com/api/id/#get-payment-method
- Transaction request: https://docs.duitku.com/api/id/#permintaan-transaksi
- Callback: https://docs.duitku.com/api/id/#callback
- Check transaction: https://docs.duitku.com/api/id/#cek-transaksi
- Payment methods: https://docs.duitku.com/api/id/#metode-pembayaran
- Expiry period: https://docs.duitku.com/api/id/#expiry-period
- HTTP codes: https://docs.duitku.com/api/id/#http-code
- Repository: https://github.com/fatkulnurk/duitku-go

For repository internals, contribution rules, and the full struct reference,
read `AGENTS.md` in the repository root.
