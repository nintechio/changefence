# Policy reference

Changefence reads JSON into a conservative built-in policy. Fields present in `.changefence.json` replace the matching default; omitted fields keep it. Arrays can be set to `[]` to clear a default list. Unknown fields and malformed values are errors.

The published [JSON Schema](../policy.schema.json) provides editor completion and validation:

```json
{
  "$schema": "https://raw.githubusercontent.com/nintechio/changefence/main/policy.schema.json"
}
```

## Global fields

| Field | Default | Meaning |
|---|---:|---|
| `max_files` | `12` | Maximum changed files. `0` disables the limit. |
| `max_additions` | `500` | Maximum total added lines. `0` disables the limit. |
| `max_deletions` | `500` | Maximum total deleted lines. `0` disables the limit. |
| `max_lines_per_file` | `300` | Maximum additions plus deletions in one file. `0` disables the limit. |
| `allowed` | `[]` | If non-empty, every changed path must match at least one pattern. |
| `forbidden` | `[]` | Paths that must never change in the evaluated diff. This rule cannot be approved. |
| `protected` | governance files | Paths that produce an explicit-review violation. |
| `source_patterns` | common source roots | Patterns used to recognize source changes. |
| `test_patterns` | common test layouts | Patterns used to recognize test changes. |
| `require_tests_for_source` | `true` | Require at least one matching test-file change when source changes. |
| `allow_dependency_changes` | `false` | Permit recognized dependency manifest and lockfile changes. |
| `allow_binary_changes` | `false` | Permit binary changes, whose churn cannot be measured by Git numstat. |
| `zones` | `[]` | Independent limits for selected repository areas. |

The default protected paths are:

```text
.changefence.json
.github/workflows/**
CODEOWNERS
SECURITY.md
```

Patterns without a slash match a basename at any depth, so `CODEOWNERS` also matches `.github/CODEOWNERS`.

The recognized dependency basenames cover common JavaScript, Python, Go, Rust, Ruby, PHP, Java/Gradle, Elixir, Swift, and .NET manifests and lockfiles. They include wildcard families such as `requirements*.txt` and `*.csproj`. Detection is intentionally conservative; open an issue when an ecosystem file is missing.

## Glob syntax

Patterns are repository-relative and case-sensitive. Changefence uses predictable path-segment matching:

- `*` matches characters within one path segment.
- `?` matches one character within a segment.
- `[abc]` and `[a-z]` match character classes.
- `**` as a complete segment matches zero or more path segments.
- A pattern without `/` matches the changed path's basename.

Examples:

| Pattern | Matches | Does not match |
|---|---|---|
| `src/**` | `src/app.go`, `src/api/v1/app.go` | `source/app.go` |
| `**/tests/**` | `tests/a.go`, `packages/api/tests/a.go` | `packages/api/testdata/a.go` |
| `app/**/generated/*.go` | `app/api/v1/generated/types.go` | `app/generated/nested/types.go` |
| `*.lock` | `poetry.lock`, `web/yarn.lock` | `web/yarn.lock.backup` |

Brace expansion, extglobs, and `!` negation are not supported. Use `allowed` and `forbidden` together instead of negative patterns.

## Zones

Zones apply a second set of limits to every changed file matching their patterns. Zones may overlap; all matching zones are evaluated.

```json
{
  "zones": [
    {
      "name": "production-infrastructure",
      "patterns": ["infra/prod/**"],
      "max_files": 3,
      "max_additions": 120,
      "max_deletions": 80,
      "max_lines_per_file": 80,
      "protected": true
    }
  ]
}
```

Each zone requires a unique non-empty `name` and at least one pattern. Its numeric limits default to `0`, which disables that zone limit. `protected: true` emits one `zone-protected` violation for every matching path.

## Rules and violation names

| Rule | Trigger |
|---|---|
| `max-files` | Global changed-file budget exceeded. |
| `max-additions` | Global addition budget exceeded. |
| `max-deletions` | Global deletion budget exceeded. |
| `max-lines-per-file` | Per-file churn budget exceeded. |
| `outside-allowed` | A path matches no `allowed` pattern. |
| `forbidden` | A path matches a `forbidden` pattern. Non-approvable. |
| `protected` | A path matches a `protected` pattern. |
| `dependency-change` | A recognized dependency file changed while disallowed. |
| `binary-change` | A binary file changed while disallowed. |
| `tests-required` | Source changed without a matching test change. |
| `zone-max-files` | Zone file budget exceeded. |
| `zone-max-additions` | Zone addition budget exceeded. |
| `zone-max-deletions` | Zone deletion budget exceeded. |
| `zone-max-lines-per-file` | Zone per-file churn budget exceeded. |
| `zone-protected` | A protected zone contains a changed path. |

## Trusted CLI approvals

The CLI can record narrow, explicit exceptions. Approved violations remain in every report; they are marked `approved: true` and no longer make the result fail.

```bash
# all violations of one rule
changefence -base origin/main -approve protected

# one rule under a path pattern
changefence -base origin/main -approve 'protected:docs/generated/**'

# one zone's aggregate rule
changefence -base origin/main -approve 'zone-max-files@documentation'

# one rule, zone, and path pattern
changefence -base origin/main -approve 'zone-protected@migrations:db/migrations/**'
```

`-approve` is repeatable. The `forbidden` rule cannot be approved.

Approvals are absent from the pull-request Action on purpose. Do not place an approval in a workflow definition that the pull request itself can modify. Use repository ruleset bypasses, a separately controlled required workflow, or another trusted orchestration layer. See the [security model](security-model.md).

## Policy source selection

The default `-policy-source auto` behavior is:

- normal Git operation: read the policy from the base commit;
- `-numstat` offline operation: read the policy from the worktree.

`-policy-source base` and `-policy-source worktree` are available explicitly. CI should use `base`. Worktree mode is intended for policy authoring and offline fixtures, not enforcement of an untrusted pull request.
