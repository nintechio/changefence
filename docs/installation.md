# Installation and rollout

## GitHub Action

Changefence ships as a Docker container Action and therefore requires a Linux runner.

```yaml
name: Changefence
on: [pull_request]

permissions:
  contents: read

jobs:
  scope:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0
      - uses: nintechio/changefence@v1
```

The Action has no configuration inputs. It always reads `.changefence.json` from the pull request's base commit and emits GitHub annotations. Keeping the enforcement surface fixed prevents a proposed workflow from passing a weaker base, another policy path, or its own approval selector into the Action.

Use a full release tag such as `nintechio/changefence@v1.0.0`, or an immutable commit SHA, when your supply-chain policy does not permit a moving major tag.

### Safe rollout

1. Create one focused setup pull request containing `.changefence.json` and the workflow.
2. Review the policy limits against representative recent pull requests.
3. Merge the setup change before marking the check as required. The policy cannot be read from the base branch until it exists there.
4. Open a small test pull request that should pass and a deliberate boundary-crossing change that should fail.
5. Add the check to a repository or organization ruleset.
6. For high-assurance use, make it a required workflow managed outside the repository's proposed revision.

When policy is absent from the base, Changefence uses its conservative defaults and reports `defaults (policy absent from base)` as the source. It never reads a newly proposed policy as trusted configuration.

## Go install

Changefence supports Go 1.22 and newer:

```bash
go install github.com/nintechio/changefence@latest
changefence -version
```

Pin a version for repeatable tool bootstrap:

```bash
go install github.com/nintechio/changefence@v1.0.0
```

## Release archives

Releases contain static binaries for Linux, macOS, and Windows on amd64 and arm64. Every release includes SHA-256 checksums and a GitHub/Sigstore build-provenance attestation.

After downloading an archive and `checksums.txt`:

```bash
sha256sum --check --ignore-missing checksums.txt
gh attestation verify changefence_1.0.0_linux_amd64.tar.gz \
  --repo nintechio/changefence
```

On macOS, use `shasum -a 256` to compare the printed digest when GNU `sha256sum` is unavailable.

## Local policy authoring

Normal operation reads the policy from the base commit. While creating or editing policy locally, opt into the worktree copy:

```bash
changefence \
  -base origin/main \
  -policy-source worktree \
  -config .changefence.json
```

Switch back to the default base-policy behavior before reproducing CI:

```bash
changefence -base origin/main
```

## Offline numstat input

The engine can evaluate a captured Git numstat stream without running Git itself:

```bash
git diff --numstat -z --no-renames origin/main...HEAD > diff.numstat
changefence \
  -numstat diff.numstat \
  -config .changefence.json \
  -format json
```

With `-numstat`, automatic policy selection uses the worktree policy because no repository base is required. File status defaults to `modified`; numstat alone does not carry add/delete status.

## Container image locally

Build the same container used by the Action:

```bash
docker build -t changefence:local .
docker run --rm \
  -v "$PWD:/repo:ro" \
  -w /repo \
  changefence:local -base origin/main
```

The container runs as a non-root user and needs read access to the checkout and its Git object database.
