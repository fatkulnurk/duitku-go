# Security Policy

## Reporting a vulnerability

Do not open a public issue for a security problem.

Report it privately through GitHub's
[security advisories](https://github.com/fatkulnurk/duitku-go/security/advisories/new),
or email the maintainer listed in `go.mod` and the repository profile.

Please include:

- A description of the issue and its impact.
- Steps to reproduce, or a proof of concept.
- The affected version or commit.
- Any suggested fix.

You can expect an acknowledgement within a few days. Please allow time for a fix
and a release before disclosing publicly.

## Scope

This library handles payment signing and callback authentication, so the
following are security relevant:

- **Signature generation.** Every request is signed with
  `HMAC_SHA256(stringToSign, apiKey)`, lowercase hex, with the API key as the
  HMAC key. A change that reintroduces MD5, plain SHA256, or any scheme where
  the API key is concatenated onto the message instead of used as the HMAC key
  is a vulnerability.
- **Signature verification.** Callback verification must use `hmac.Equal`. Using
  `==` or `strings.EqualFold` reintroduces a timing side channel and accepts
  mixed case hex that the spec forbids.
- **Callback authenticity.** `ParseCallback` must reject a callback whose
  `merchantCode` does not match the configured one, and must reject a missing or
  empty signature before verification.
- **Credential handling.** `APIKey` must never be logged. The
  `LogEveryRequestAndResponse` option logs request and response bodies; it must
  not be enabled by default.
- **Error leakage.** Handler errors must be logged, not returned to the caller.

## Known limitations, not vulnerabilities

These are documented properties of Duitku's API. They are not bugs in this
library.

- **`resultCode` is not covered by the callback signature.** The signature is
  computed over `merchantCode + amount + merchantOrderId`, so a captured
  callback can have its `resultCode` altered without invalidating the HMAC.
  Confirm the final state with `CheckTransaction` before releasing goods or
  credit. This is called out in the `IsSuccessful` documentation.
- **`returnUrl` is user controlled.** Duitku states that the customer can edit
  the redirect URL, so the redirect `resultCode` must never be used to change a
  payment status.
- **Duitku does not sign the callback source IP.** Duitku publishes its
  outbound IP ranges for allow listing; that is an infrastructure control, not
  something this library can enforce.

## Supported versions

The latest released minor version receives security fixes.
