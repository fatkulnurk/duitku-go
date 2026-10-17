# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Full alignment with the current Duitku API v2 documentation. This release
contains breaking changes; see the migration notes below.

### Fixed

- **Signatures now use HMAC-SHA256.** Every request was previously signed with
  an obsolete scheme: a single MD5 or plain SHA256 digest over the parameters
  with the API key appended to the message. Duitku requires
  `HMAC_SHA256(stringToSign, apiKey)` with the API key as the HMAC key, encoded
  as lowercase hex. The old signatures were always rejected by Duitku.
  Affects `CreateTransaction`, `CheckTransaction`, `GetPaymentMethods` and
  callback verification.
- **Callback verification is now constant time.** The comparison used
  `strings.EqualFold`, which is case insensitive and short circuits. It now uses
  `hmac.Equal`.
- **Callback `customerName` is now parsed.** The documented field, sent for Nobu
  QRIS (`NQ`) and Nusapay QRIS (`SQ`), was silently dropped.
- **`PaymentMethodQrisNobu` corrected from `QN` to `NQ`.**
- **`PaymentMethodIndodanaPaylater` corrected from `ID` to `DN`.**
- **API errors are reported accurately.** The error decoder read
  `responseCode`/`responseMessage`, but Duitku sends a capital-M `Message` key
  for HTTP 400/401/404/409, so the reason was reported as empty. A non-JSON or
  empty error body was also reported as a JSON decoding error; it is now
  reported as an HTTP failure with the status and a truncated body.
- **`TransactionResponse` gained the `AppURL` field** for e-commerce channels.
- **`amount` decodes whether Duitku sends a string or a number**, via the new
  `StringOrInt` type.
- **`expiryPeriod` is omitted when zero** instead of being sent as `0`.
- **Empty `MerchantCode` or `APIKey` now fails fast** with a clear error instead
  of sending a request signed with an empty key.
- **A custom `HTTPClient` with no timeout gets a 30 second default** instead of
  blocking forever.
- **Response bodies are always drained and closed**, so connections can be
  reused.
- **`ParseCallback` reads only the POST body**, so query string parameters can
  no longer influence signed fields, and it rejects a callback whose
  `merchantCode` does not match the configured merchant code.
- **`HandleCallback` no longer echoes internal handler errors** to the caller.
  The error is logged and the response is a plain 500.

### Added

- `TransactionRequest.Validate()`: opt-in validation of the documented rules
  (minimum amount, field lengths, item totals matching `paymentAmount`,
  per-channel requirements).
- `ExpiryPeriodFor()` and `ValidateExpiryPeriod()` with the documented expiry
  window for every channel.
- `APIError` type carrying the HTTP status, the Duitku code, the message and the
  truncated raw body.
- `CallbackData.IsFailed()`.
- Redirect status constants: `RedirectStatusSuccess`,
  `RedirectStatusPending`, `RedirectStatusCanceled`.
- `StringOrInt` for values Duitku may send as either a string or a number.
- Tokopedia constants: `PaymentMethodTokopediaCard`,
  `PaymentMethodTokopediaEWallet`, `PaymentMethodTokopediaOthers`.
- Documented limits: `MinimumPaymentAmount`, `MaximumMerchantOrderIDLength`,
  `MaximumProductDetailsLength`.
- AI and contributor documentation: `AGENTS.md`, `SKILL.md`, `CLAUDE.md`,
  `llms.txt`, `.github/copilot-instructions.md`, `.cursor/rules/duitku-go.mdc`.

### Removed

- `VoidTransaction` and its request and response types. The endpoint it called
  is not part of the documented API v2.
- `TransactionRequest.IsSubscription`, `TransactionRequest.SubscriptionDetail`
  and the `SubscriptionDetail` type. Duitku removed the subscription feature
  from the inquiry request in November 2025.
- The `FrequencyDaily`, `FrequencyWeekly`, `FrequencyMonthly` and
  `FrequencyYearly` constants, which belonged to the removed subscription
  feature.
- `PaymentMethodQrisDana` (`DQ`) and `PaymentMethodQrisGudangVoucher` (`GQ`).
  Duitku retired those channels in August 2025 and September 2026 respectively.

### Changed

- `TransactionResponse.Amount` and `TransactionStatusResponse.Amount` are now
  `StringOrInt` instead of `string`. Use `.String()` or `.Int()`.
- `PaymentMethodJeniusPay` is grouped under e-banking, matching the docs.
- `go.mod` requires Go 1.22.
- CI runs `go test -race`, checks `gofmt`, pins the `staticcheck` version, and
  runs the formatter and cache steps in a useful order.
- Test suite rewritten to be table driven throughout, asserting fixed signature
  vectors rather than recomputing the expected value with the helper under test.
- The `.gitignore` no longer ignores `.github/workflows/`, and the stray
  Markdown code fences were removed.

### Migration from 0.x

1. **Remove any `VoidTransaction` call.** There is no documented replacement in
   API v2.
2. **Remove `IsSubscription` and `SubscriptionDetail` from your requests.**
3. **Update stored payment method codes:** `QN` becomes `NQ`, `ID` becomes
   `DN`. Remove any use of `DQ` or `GQ`.
4. **Update `amount` handling.** Replace `response.Amount == "40000"` with
   `response.Amount.String() == "40000"` or use `response.Amount.Int()`.
5. **Signatures change automatically.** No action needed, but be aware that
   requests which previously appeared to succeed against a mock will now produce
   different signatures, which is correct.
