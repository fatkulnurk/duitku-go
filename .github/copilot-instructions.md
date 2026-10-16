# Copilot Instructions

Repository instructions live in [AGENTS.md](../AGENTS.md). Read it before
proposing changes. This file summarises the non negotiable points.

## Project

`github.com/fatkulnurk/duitku-go` is a Go client for the Duitku Payment Gateway
API v2. It is a single flat package named `duitku`.

## Hard rules

1. **Standard library only.** No third party imports. `go.mod` has no `require`
   block and there is no `go.sum`.
2. **HMAC-SHA256 for all signatures.** Use `hmac.New(sha256.New, []byte(apiKey))`
   and encode with `hex.EncodeToString`. Never use `md5`, and never use
   `sha256.Sum256` over a message with the API key appended.
3. **Constant-time signature verification.** Use `hmac.Equal`. Never `==` and
   never `strings.EqualFold`.
4. **All tests are table driven.** Each `TestXxx` declares a `tests` slice with
   named cases and loops with `for _, tt := range tests`.
5. **Signature tests assert fixed literals.** Do not recompute the expected
   value with the helper under test.
6. **Tests never hit the network.** Use `httptest.NewServer` or inject an
   `http.RoundTripper`.

## Before submitting

```bash
gofmt -l .          # must print nothing
go build ./...      # must exit 0
go vet ./...        # must exit 0
go test ./...       # must pass
```

## Signature stringToSign per operation

| Operation | stringToSign |
|---|---|
| Get Payment Method | `merchantcode + amount + datetime` |
| Create Transaction | `merchantCode + merchantOrderId + paymentAmount` |
| Check Transaction | `merchantCode + merchantOrderId` |
| Callback | `merchantCode + amount + merchantOrderId` |

## Do not reintroduce

- `VoidTransaction` (not in the documented API v2)
- `isSubscription` / `subscriptionDetail` (removed by Duitku in November 2025)
- Payment codes `DQ` and `GQ` (channels retired)
- Payment codes `QN` and `ID` (the correct codes are `NQ` and `DN`)

## Known Duitku quirks that are intentional

- Get Payment Method uses the lowercase key `merchantcode`; all other endpoints
  use `merchantCode`.
- `amount` arrives as a string in some responses and a number in others; use
  `StringOrInt` and its `.String()` / `.Int()` accessors.
- Callback `resultCode` is not part of the signed string.
- Callback `01` is Failed, but redirect `01` is Pending and `02` is Canceled.
- `expiryPeriod` 0 means "use the channel default".
