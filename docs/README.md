# smokeforge documentation

This directory describes how `smokeforge` is designed, how the registry works,
and how to develop and test the CLI. Documents distinguish **implemented**
behavior from **planned** behavior where relevant.

## Reading order

New to the project? Read in this order:

1. [Root README](../README.md) — overview, supported stacks/features, quick start.
2. [Visual architecture](architecture.md) — Mermaid diagrams of the system, flows, and trust boundaries.
3. [Architecture overview](architecture/overview.md) — components and data flow.
4. [CLI and lifecycle](architecture/cli.md) — commands, wizard, project lifecycle.
5. [Registry manifest](registry/manifest.md) — registry schema and validation.
6. [Overlay and patch model](registry/overlays.md) — how features compose.
7. [Project state](architecture/project-state.md) — lockfiles, hashes, recovery.
8. [Testing](development/testing.md) — verification strategy.
9. [Release guide](development/release.md) — versioning and publishing.
10. [Security](development/security.md) — threat model and controls.
11. [Cross-platform compatibility](development/cross-platform.md) — supported matrix and rules.

## Structure

```text
docs/
  README.md                        this index
  architecture.md                  visual architecture with Mermaid diagrams
  architecture/
    overview.md                    components, boundaries, generation pipeline
    cli.md                         commands, wizard, flags, lifecycle commands
    generation-engine.md           resolver, renderer, patcher, staging
    project-state.md               .smokeforge.json, lockfile, drift, rollback
    fullstack.md                   multi-target monorepo composition
  registry/
    manifest.md                    registry repo layout and manifest reference
    overlays.md                    anchors, patch operations, authoring guide
  development/
    setup.md                       prerequisites, build, test, lint, conventions
    testing.md                     test pyramid, contract/golden/build matrix
    release.md                     versioning, snapshot, tagging, Homebrew, registry
    security.md                    threat model, controls, verification, gaps
    cross-platform.md              OS/arch matrix, rules, platform notes
    roadmap.md                     milestones, scope, out-of-scope
  superpowers/
    specs/                         approved design records
    plans/                         implementation plans
```

## Status legend

| Term | Meaning |
| --- | --- |
| **Implemented** | Code exists and is covered by tests in this repository. |
| **In progress** | Actively being implemented in the current milestone. |
| **Planned** | Designed and approved; not yet implemented. |
| **Future** | Explicitly out of v1; may be considered later. |

## Design records

The authoritative decisions, including rejected alternatives, live in
[specs/2026-11-12-smokeforge-design.md](superpowers/specs/2026-11-12-smokeforge-design.md).
Maintainer release steps: [PUBLISHING.md](../PUBLISHING.md).
Security reporting: [SECURITY.md](../SECURITY.md).

Implementation sequencing lives in:

- [Foundation plan](superpowers/plans/2026-11-12-smokeforge-foundation.md)
- [Generation plan](superpowers/plans/2026-11-12-smokeforge-generation.md)
- [Lifecycle plan](superpowers/plans/2026-11-12-smokeforge-lifecycle.md)
