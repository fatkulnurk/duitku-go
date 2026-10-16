# CLAUDE.md

See [AGENTS.md](./AGENTS.md) for all repository instructions, the full API
reference, signature vectors, and invariants.

Quick reference for this repository:

- Go client for the Duitku Payment Gateway API v2. Single flat package `duitku`.
- **Standard library only.** No third party imports, no `go.sum`.
- **HMAC-SHA256 signatures.** `hmac.New(sha256.New, []byte(apiKey))`, lowercase
  hex. Never MD5, never `sha256.Sum256(msg + apiKey)`.
- Verify callbacks with `hmac.Equal`, never `==` or `strings.EqualFold`.
- All tests are table driven and loop over a `tests` slice.
- Signature tests assert fixed literals, not values recomputed by the same
  helper.

Before claiming a task is done:

```bash
gofmt -l .          # must print nothing
go build ./...      # must exit 0
go vet ./...        # must exit 0
go test ./...       # must pass
```

Read `AGENTS.md` before making changes.
