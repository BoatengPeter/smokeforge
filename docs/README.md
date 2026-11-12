# smokeforge documentation

This directory describes how `smokeforge` is designed, how the registry works,
and how to develop and test the CLI. Documents distinguish **implemented**
behavior from **planned** behavior where relevant.

## Reading order

New to the project? Read in this order:

1. [Root README](../README.md) — overview, supported stacks/features, quick start.
2. [Architecture overview](architecture/overview.md) — components and data flow.
3. [CLI and lifecycle](architecture/cli.md) — commands, wizard, project lifecycle.
4. [Registry manifest](registry/manifest.md) — registry schema and validation.
5. [Overlay and patch model](registry/overlays.md) — how features compose.
6. [Project state](architecture/project-state.md) — lockfiles, hashes, recovery.
7. [Testing](development/testing.md) — verification strategy.

## Structure

```text
docs/
  README.md                        this index
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
Implementation sequencing lives in
[plans/2026-11-12-smokeforge-foundation.md](superpowers/plans/2026-11-12-smokeforge-foundation.md).
