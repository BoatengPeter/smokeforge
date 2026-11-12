# smokeforge M4 — Project lifecycle

> **For agentic workers:** Use superpowers:test-driven-development for every production behavior.

**Status:** complete (lifecycle scope). M3 registry content remains deferred.

**Goal:** `smokeforge add` and `smokeforge remove` safely evolve an existing
generated project, with drift detection, backups, rollback, and project-aware
`doctor` checks.

## Scope

In M4 lifecycle:

- `add <feature[:variant]>...` against the project's locked registry.
- `remove <feature[:variant]>...` for reversible overlays.
- Drift detection against recorded SHA-256 hashes.
- File backups under `.smokeforge/backups/` and automatic restore on failure.
- State/lock updates after successful mutation.
- `doctor` project-state integrity check.
- Tests with the neutral fixture registry.

Out of scope (still planned): fullstack/multi-target, `install`, client API
wiring, official stack templates, dry-run for mutations.

## Reversibility

Automatic removal supports overlays containing only:

- `addFiles` (delete files if untouched),
- `insertBefore`/`insertAfter` (excise marked blocks),
- `env` (remove generated keys from env files).

Overlays containing `patchJSON`, `patchYAML`, `patchTOML`, or `addDependency`
are refused with a manual-removal message. This is conservative until inverse
operation records exist.

## Mutation transaction

```text
load state + lock
  → resolve new/removed overlays against locked registry
  → refuse on drift (add: `--force` may override; remove: always strict)
  → snapshot impacted files to .smokeforge/backups/<timestamp>/
  → apply or revert overlays
  → update env files
  → rehash tracked files for every applied overlay
  → update state + lock
  → discard backup
  → on any error: restore backup, leave state/lock unchanged
```

## State additions

`AppliedOverlay` gains optional `env` keys so removal knows which variables were
provided by the overlay. Env files (`.env`, `.env.example`) are included in the
hash map for new generations.

## Tasks

- [x] Plan and fixture feature (`metrics`) for add/remove tests
- [x] `project`: `FindRoot`, `CheckDrift`, `CreateBackup`/restore, `Rehash`
- [x] `patch`: `removeBlock`, `CheckRemovable`, `Remove`
- [x] `engine`: `Add`, `Remove`, shared transaction helpers
- [x] `cli`: `add`, `remove`, project-aware `doctor`
- [x] Docs, verification, commits
