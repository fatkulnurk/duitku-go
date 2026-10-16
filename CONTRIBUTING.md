# Contributing

Thanks for considering a contribution to `duitku-go`.

## Before you start

Read [AGENTS.md](./AGENTS.md). It documents the full API surface, the
invariants, and the Duitku quirks that shape this codebase. It is written for
both humans and AI agents.

## Ground rules

1. **Standard library only.** No third party imports, ever. `go.mod` has no
   `require` block and there is no `go.sum`.
2. **HMAC-SHA256 signatures only.** Use `hmac.New(sha256.New, []byte(apiKey))`.
   MD5 and plain SHA256 with the API key appended are obsolete.
3. **Constant-time verification.** Use `hmac.Equal`, never `==` or
   `strings.EqualFold`.
4. **Table driven tests.** Every `TestXxx` declares a `tests` slice with named
   cases and loops over it. Add a case rather than a new test function where it
   fits.
5. **Assert fixed values in signature tests.** Do not recompute the expected
   value with the helper under test; that passes even when the algorithm is
   wrong.
6. **No network in tests.** Use `httptest.NewServer` or inject an
   `http.RoundTripper`.

## Development

```bash
git clone https://github.com/fatkulnurk/duitku-go
cd duitku-go

gofmt -l .              # must print nothing
go build ./...
go vet ./...
go test ./...
go test -cover ./...    # expect ~99% statement coverage
go test -race ./...     # requires cgo and a C compiler
```

## Making a change

1. Create a branch: `git checkout -b fix/short-description`.
2. Make the change, including tests.
3. Update `README.md`, `doc.go`, `AGENTS.md` and `CHANGELOG.md` if the public
   API or documented behaviour changed.
4. Run the full command list above.
5. Open a pull request describing the problem, the fix, and how you verified it.

## Adding a payment method

1. Add the constant to the correct category block in `constants.go`.
2. Add its expiry window to `expiryPeriodLimits` in `expiry.go` if the docs
   publish one.
3. Extend `validateChannelRequirements` in `validation.go` if the channel has
   extra required parameters.
4. Add it to the tables in `doc.go`, `README.md` and `SKILL.md`, and to the
   dropdown in `example/main.go`.
5. Add cases to `expiry_test.go` and `validation_test.go`.

## Reporting a bug

Include the Go version, the SDK version, the payment method involved, the
request you sent with credentials redacted, and the full error. If the error is
an `*APIError`, include `StatusCode`, `Code` and `Message`. You can also set
`LogEveryRequestAndResponse: true` in `Config` to capture the raw exchange.

For security issues, follow [SECURITY.md](./SECURITY.md) instead of opening a
public issue.
