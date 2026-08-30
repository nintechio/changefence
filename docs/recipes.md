# Policy recipes

These examples are starting points, not universal risk scores. Tune limits from real pull-request history, then make each change to policy a reviewed governance decision.

## Small application repository

```json
{
  "$schema": "https://raw.githubusercontent.com/nintechio/changefence/main/policy.schema.json",
  "max_files": 15,
  "max_additions": 600,
  "max_deletions": 400,
  "max_lines_per_file": 300,
  "allowed": ["src/**", "tests/**", "docs/**", "README.md"],
  "forbidden": ["secrets/**", "backups/**"],
  "protected": [".changefence.json", ".github/workflows/**", "CODEOWNERS", "SECURITY.md"],
  "source_patterns": ["src/**"],
  "test_patterns": ["tests/**", "**/*_test.*", "**/*.test.*"],
  "require_tests_for_source": true,
  "allow_dependency_changes": false,
  "allow_binary_changes": false
}
```

## Monorepo with package budgets

Global limits stop repository-wide sprawl. Zones add tighter budgets for independently owned packages:

```json
{
  "max_files": 40,
  "max_additions": 1500,
  "max_deletions": 1000,
  "source_patterns": ["packages/*/src/**", "services/*/src/**"],
  "test_patterns": ["packages/*/tests/**", "services/*/tests/**", "**/*_test.*"],
  "require_tests_for_source": true,
  "zones": [
    {
      "name": "billing-service",
      "patterns": ["services/billing/**"],
      "max_files": 12,
      "max_additions": 500,
      "max_lines_per_file": 220
    },
    {
      "name": "shared-auth",
      "patterns": ["packages/auth/**"],
      "max_files": 8,
      "max_additions": 300,
      "protected": true
    }
  ]
}
```

Zones may overlap. A file in two zones must satisfy both.

## Database migrations

Migration changes are often valid but deserve a deliberately small blast radius:

```json
{
  "zones": [
    {
      "name": "database-migrations",
      "patterns": ["db/migrations/**", "migrations/**"],
      "max_files": 2,
      "max_additions": 250,
      "max_deletions": 20,
      "max_lines_per_file": 180,
      "protected": true
    }
  ]
}
```

The protected zone is deliberately noisy: it makes the migration visible in the report even when its size is acceptable.

## Infrastructure repository

Use `forbidden` for areas that this check must never accept, and `protected` for areas that a trusted ruleset may override after review:

```json
{
  "max_files": 10,
  "max_additions": 300,
  "max_deletions": 200,
  "allowed": ["modules/**", "environments/dev/**", "environments/staging/**", "docs/**"],
  "forbidden": ["environments/prod/**", "state/**", "*.tfstate", "*.tfstate.backup"],
  "protected": [".changefence.json", ".github/workflows/**", "CODEOWNERS"],
  "allow_dependency_changes": false,
  "allow_binary_changes": false
}
```

`forbidden` violations cannot be approved with the CLI.

## Documentation automation

Generated documentation can be broad while handwritten documentation should remain focused. Use overlapping zones to express that difference:

```json
{
  "allowed": ["docs/**", "README.md"],
  "source_patterns": [],
  "require_tests_for_source": false,
  "zones": [
    {
      "name": "handwritten-docs",
      "patterns": ["docs/**"],
      "max_files": 15,
      "max_additions": 800,
      "max_lines_per_file": 300
    },
    {
      "name": "generated-reference",
      "patterns": ["docs/generated/**"],
      "max_files": 100,
      "max_additions": 0,
      "max_lines_per_file": 0
    }
  ]
}
```

Because both zones match `docs/generated/**`, the tighter handwritten-docs limits still apply. Exclude generated files from the broader zone by choosing non-overlapping directory layouts, or make the global/documentation limits large enough for both. Changefence does not support negative glob patterns.

## Dependency update bots

Dependency changes are disabled by default. If a repository intentionally accepts bot-driven manifest updates, allow them while keeping a small dedicated zone:

```json
{
  "allow_dependency_changes": true,
  "zones": [
    {
      "name": "dependencies",
      "patterns": ["package.json", "package-lock.json", "go.mod", "go.sum", "pyproject.toml", "uv.lock"],
      "max_files": 4,
      "max_additions": 400,
      "max_deletions": 400,
      "protected": true
    }
  ]
}
```

This checks scope only. Continue to use dependency review, vulnerability scanning, lockfile integrity checks, and ordinary tests.
