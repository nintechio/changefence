# Contributing

Changefence should remain small, deterministic, explainable, and useful without a hosted service. Contributions are welcome when they preserve those properties and solve a demonstrated workflow problem.

## Before opening a pull request

For a bug, open an issue with a minimal policy, numstat or repository shape, actual output, and expected output. Security issues belong in a private advisory as described in [SECURITY.md](SECURITY.md).

For a new rule, describe:

1. the concrete unsafe or wasteful change it catches;
2. why existing rules cannot express it;
3. the false-positive risk;
4. how it remains deterministic across platforms;
5. the proposed report and exit behavior.

Features that require source upload, telemetry, an account, or a model are outside the core project's direction.

## Development

Changefence requires Go 1.22 or newer and has no third-party Go dependencies.

```bash
go test ./...
go test -race ./...
go vet ./...
go build -trimpath -buildvcs=false ./...
```

To exercise the release packaging on Linux, matching the release runner:

```bash
./scripts/build-release.sh 0.0.0-test /tmp/changefence-release
```

Keep behavior changes covered by focused unit tests and, when Git trust behavior changes, a temporary-repository integration test. Preserve complete failure output when reporting a failing command.

To exercise the CLI against its fixture:

```bash
go run . \
  -numstat testdata/demo.numstat \
  -config .changefence.example.json \
  -format json
```

## Pull requests

- Keep each pull request focused on one behavior or documentation goal.
- Update the policy schema and reference documentation with configuration changes.
- Document security assumptions, not just happy-path usage.
- Do not add generated assets when a small source SVG or text fixture is sufficient.
- Avoid new dependencies unless the maintenance and supply-chain cost is clearly justified.
- Add a changelog entry for user-visible behavior after the first release.

By contributing, you agree that your contribution is licensed under the repository's MIT License and that you have the right to submit it.
