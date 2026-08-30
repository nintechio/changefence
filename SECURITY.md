# Security policy

## Supported versions

Security fixes are provided for the latest stable major release. Users should update to the newest published patch version before reporting a problem that may already be fixed.

| Version | Supported |
|---|---|
| Latest `1.x` | Yes |
| Pre-release and development snapshots | Best effort |

## Reporting a vulnerability

Use this repository's **Security** tab to open a private GitHub Security Advisory. Do not include exploit details, credentials, private repository content, or personal data in a public issue.

Please include:

- affected Changefence version or commit;
- operating system and installation method;
- a minimal repository or policy shape when safe to share;
- expected trust boundary and observed bypass;
- reproduction steps and impact;
- any proposed mitigation.

We aim to acknowledge a report within 72 hours. Validation and remediation timelines depend on severity and reproducibility. We will coordinate disclosure and credit with the reporter unless anonymity is requested.

## In scope

- bypassing base-policy trust or GitHub event-base selection;
- accepting malformed policy in a permissive way;
- command, workflow-annotation, path, or archive injection;
- approval-selector bypass, including approval of a forbidden path;
- unsafe container permissions or release provenance problems;
- materially incorrect policy results with a security impact.

Semantic code-quality failures in a changed project are outside Changefence's scope; it enforces diff boundaries rather than proving code correctness. See the full [security model](docs/security-model.md).
