<div align="center">

# changefence

**A hard scope boundary for automated code changes.**

[![CI](https://github.com/nintechio/changefence/actions/workflows/ci.yml/badge.svg)](https://github.com/nintechio/changefence/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.22%2B-00add8.svg)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-8b5cf6.svg)](LICENSE)

</div>

Coding agents are excellent at making a requested change—and sometimes five unrequested ones. `changefence` turns review expectations into an executable diff policy. It is deterministic, model-free, and works with changes made by humans or any agent.

```bash
changefence -base origin/main
```

```text
changefence: 4 files, +86/-12
  FAIL protected            protected path requires separate human review (.github/workflows/ci.yml)
  FAIL tests-required       source changed without a matching test-file change
```

Exit `0` means the change is inside the declared boundary; exit `1` means it crossed one. Exit `2` means the policy or repository could not be evaluated.

## Policies that ordinary linters cannot express

- maximum files, total additions/deletions, and churn per file
- allowed roots and absolutely forbidden paths
- protected governance files that always require human review
- dependency-manifest changes disabled unless explicitly authorized
- source changes must be accompanied by test changes
- JSON output for bots and CI summaries

Copy [`.changefence.example.json`](.changefence.example.json) to `.changefence.json` and make the boundary fit the task. Missing configuration uses conservative defaults; malformed configuration fails closed.

## GitHub Actions

```yaml
name: Change fence
on: [pull_request]
jobs:
  scope:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v6
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v7
        with:
          go-version: '1.22'
          cache: false
      - run: go run github.com/nintechio/changefence@latest -base '${{ github.event.pull_request.base.sha }}'
```

## Non-goals

`changefence` does not decide whether code is correct, infer intent with an LLM, or replace review. It enforces the explicit boundary *before* a reviewer spends attention on correctness.

## Development

```bash
go test ./...
go vet ./...
go build -buildvcs=false ./...
```

## License

MIT © [Nintech Ltd](https://nintech.io)
