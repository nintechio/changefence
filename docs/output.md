# Output and automation

Changefence writes reports to standard output and diagnostics to standard error. All output modes use the same evaluated report and exit codes.

## Exit codes

| Code | Meaning |
|---:|---|
| `0` | The diff is within policy, including any explicitly approved CLI violations. |
| `1` | At least one unapproved policy violation exists. |
| `2` | Input could not be evaluated safely: invalid policy, missing base, malformed Git data, unsupported option, or invalid GitHub event. |

## Text

`-format text` is the default for terminals:

```text
changefence 1.0.0: FAIL · 3 files · +86/-12
policy: .changefence.json (base:8e14bc02f83a)
  FAIL  protected                 protected path requires separate human review (.github/workflows/deploy.yml)
  FAIL  tests-required            source changed without a matching test-file change
```

## JSON

`-format json` is intended for programs. Paths and strings are encoded with Go's standard JSON encoder.

```json
{
  "version": "1.0.0",
  "base": "8e14bc02f83a4b7f000000000000000000000000",
  "policy_path": ".changefence.json",
  "policy_source": "base:8e14bc02f83a",
  "files": [
    {
      "path": "src/auth.go",
      "status": "modified",
      "additions": 42,
      "deletions": 8,
      "binary": false
    }
  ],
  "additions": 42,
  "deletions": 8,
  "violations": [
    {
      "rule": "tests-required",
      "message": "source changed without a matching test-file change",
      "approved": false
    }
  ],
  "passed": false
}
```

File statuses are `added`, `modified`, `deleted`, or `type-changed` when Changefence reads Git directly. Unknown Git status codes are preserved in lowercase. Offline numstat input reports `modified` because it has no status channel.

Violations may also include `path` and `zone`. `violations` remains present as an empty array on a clean report. Approved violations remain in the list with `approved: true` for auditability.

Within major version 1, existing JSON fields and meanings are treated as stable. New optional fields may be added in minor releases; consumers should ignore fields they do not recognize.

## Markdown

`-format markdown` produces a heading, summary, and violation table suitable for a check summary or pull-request comment.

```bash
changefence -base origin/main -format markdown >> "$GITHUB_STEP_SUMMARY"
```

## GitHub annotations

`-format github` emits workflow commands:

- unapproved path violations become `error` annotations;
- approved path violations become `notice` annotations;
- global and zone aggregate violations appear without a file location;
- `%`, carriage returns, line feeds, commas, and colons are escaped where GitHub requires it.

The published Action selects this format automatically.

## Quiet machine use

When only the decision matters, redirect output and inspect the exit code:

```bash
if changefence -base origin/main >/dev/null; then
  echo "inside scope"
else
  code=$?
  echo "changefence exited $code"
fi
```

Do not collapse exit `1` and exit `2` in enforcement code: a failed evaluation should not be interpreted as an ordinary policy rejection or a pass.
