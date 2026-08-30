## What changes?

Describe the concrete behavior or documentation problem this pull request solves.

## Evidence

Include focused tests, complete failure output, or a minimal before/after example.

## Checklist

- [ ] The change is focused and deterministic across supported platforms.
- [ ] Behavior changes have tests.
- [ ] Configuration changes update `policy.schema.json` and `docs/policy.md`.
- [ ] Security and trust-boundary effects are documented.
- [ ] `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build -trimpath -buildvcs=false ./...` pass.
