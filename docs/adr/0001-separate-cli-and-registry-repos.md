# 0001 — Separate CLI and registry repositories

- Status: Accepted
- Date: 2026-11-12

## Context

The CLI fetches template content at runtime. Template content has a different
release cadence, licensing surface, and review process from the Go CLI. It also
contains third-party agent skills.

## Decision

Keep two repositories:

- `smokeforge` — CLI, Go skills, docs, release tooling.
- `smokeforge-registry` — manifest, templates, overlays, project-scoped skills,
  contract tests, and registry releases.

One official registry is used until multi-registry/index support exists.
Registry releases are independent tagged tarballs verified by SHA-256.

## Consequences

- Schema changes can require coordinated changes in both repositories.
- Registry CI must pin a CLI version and run contract/build tests.
- Local development uses `--registry ../smokeforge-registry`.
- Per-ecosystem registry splits are deferred until size, ownership, or release
  cadence justifies them.
