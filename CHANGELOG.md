# Changelog

All notable changes to Changefence are documented here. The project follows [Semantic Versioning](https://semver.org/).

## [1.0.0] - 2026-08-30

### Added

- Trusted base-commit policy loading that prevents a pull request from weakening its own rules.
- GitHub pull-request base selection from the event payload, with fail-closed parsing.
- Global budgets for file count, additions, deletions, and per-file churn.
- Allowed, forbidden, and protected path policies with deterministic `**` glob matching.
- Source-to-test change coupling.
- Conservative dependency-manifest, lockfile, and binary-change controls.
- Overlapping policy zones with independent budgets and protection.
- Narrow trusted CLI approvals with rule, zone, and path selectors; forbidden paths remain non-approvable.
- Text, JSON, Markdown, and escaped GitHub annotation output.
- File add, modify, delete, and type-change status reporting.
- Docker-based GitHub Action running as a non-root user.
- Cross-platform release archives, checksums, and signed build-provenance attestations.
- JSON Schema, policy recipes, threat model, rollout guide, and automation reference.

[1.0.0]: https://github.com/nintechio/changefence/releases/tag/v1.0.0
