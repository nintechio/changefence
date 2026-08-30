# Security model

Changefence is a scope gate, not a code-security scanner. Its security claim is deliberately narrow: an untrusted proposed diff is evaluated against policy taken from a trusted base commit, and every verdict is derived from explicit, inspectable rules.

## Trust boundaries

Trusted inputs:

- the Git executable and repository object database provided by the runner;
- the base commit selected by GitHub's pull-request event payload, or by a trusted CLI caller;
- `.changefence.json` as stored in that base commit;
- approval selectors supplied by a trusted external orchestrator.

Untrusted inputs:

- files, filenames, line counts, and statuses in the proposed diff;
- the checked-out worktree policy during normal CI enforcement;
- pull-request-authored workflow changes and command-line arguments;
- report strings derived from paths.

For GitHub `pull_request` and `pull_request_target` events, Changefence ignores the ordinary `-base` value and takes `pull_request.base.sha` from `GITHUB_EVENT_PATH`. A missing or malformed event payload is an evaluation error, not a fallback to an untrusted revision.

## Defended scenarios

### A pull request weakens its own policy

The policy is loaded with Git object access from the resolved base commit. The changed worktree copy is never decoded for normal enforcement. The policy-file change itself also matches the default protected list.

### A pull request redirects the base argument

On GitHub pull-request events, the event payload wins over `-base`. Outside GitHub Actions, selecting a trusted base remains the caller's responsibility.

### Malformed policy creates permissive zero values

Configuration is decoded over conservative defaults with unknown-field rejection. Invalid types, malformed globs, negative limits, duplicate zones, and trailing JSON return exit `2`.

### Filenames contain spaces, Unicode, or shell characters

Git output is requested with NUL delimiters and passed directly as process arguments. Changefence does not invoke a shell. GitHub annotation control characters are escaped before output.

### Binary files evade line budgets

Git reports binary churn as `-`. Binary changes fail by default because their size cannot be represented by line-count budgets.

### A container sees a differently owned checkout

Git is invoked with the exact current directory as `safe.directory`; the container does not globally trust unrelated paths. The published Docker Action runs as a non-root user.

## GitHub workflow integrity

No tool running inside an untrusted pull-request workflow can make that workflow definition trusted. An attacker who can replace a required job with another job may be able to spoof a check name at the platform level.

For high-assurance repositories:

1. Use an organization or repository ruleset with a required workflow controlled outside the target repository's proposed revision.
2. Pin `nintechio/changefence` to a full release tag or immutable commit.
3. Give the job only `contents: read`.
4. Keep `fetch-depth: 0` on checkout.
5. Require review or a ruleset bypass for changes to policy, workflows, `CODEOWNERS`, and other governance paths.

The marketplace Action intentionally exposes no approval input. A protected-path violation should be resolved by the repository's trusted review and ruleset process, not by a string added to the same proposed workflow.

## `pull_request_target`

Changefence can derive a base from `pull_request_target`, but checking out and executing untrusted pull-request code in that event is dangerous because the workflow may have access to base-repository secrets and permissions. Prefer ordinary `pull_request` with read-only permissions. This warning applies to the surrounding workflow, not just Changefence.

## Runtime and data behavior

The binary itself:

- makes no network requests;
- sends no telemetry;
- reads the policy and Git metadata needed for the diff;
- writes only its report to standard output or standard error;
- does not directly inspect source-file contents; Git computes the requested line-count diff;
- has no third-party Go module dependencies.

The Docker image contains Git and CA certificates because Git is required to inspect the repository. No token is requested by the Action.

## Known limitations

- Scope compliance says nothing about semantic correctness or security of changed code.
- Git line counts do not represent binary size or generated-file complexity.
- Test coupling checks for a matching test-file change, not whether tests are meaningful or pass.
- Dependency detection is basename-based and may need additions for less common ecosystems.
- Rename detection is intentionally disabled; a rename is evaluated conservatively as deletion plus addition paths.
- Shallow clones that do not contain the base and merge base cannot be evaluated.
- Git may fetch missing objects when the caller deliberately uses a partial clone with a promisor remote; use a complete checkout for offline enforcement.
- A compromised runner, Git binary, base branch, or repository object database is outside this threat model.

Please report security problems privately as described in [SECURITY.md](../SECURITY.md).
