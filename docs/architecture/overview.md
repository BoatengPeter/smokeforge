# Architecture overview

**Status:** implemented components and planned components are called out inline.

## Purpose

`smokeforge` is a single Go binary that turns a set of choices — project kind,
stack(s), features, and project metadata — into a complete project tree. The
tree is produced from a remote, versioned template registry. Templates are
declarative data; the CLI owns all rendering, patching, validation, and
filesystem side effects.

## Non-goals

- Executing arbitrary code from templates.
- Acting as a package manager, build tool, or deployment system.
- Maintaining generated code after handoff (beyond `add`/`remove` overlays).

## Component map

```text
                    ┌────────────────────────────┐
                    │        cmd/smokeforge      │
                    └─────────────┬──────────────┘
                                  │
                    ┌─────────────▼──────────────┐
                    │        internal/cli        │  Cobra commands, flags
                    │        internal/tui        │  interactive wizard
                    └───────┬─────────────┬──────┘
                            │             │
          ┌─────────────────▼───┐   ┌─────▼─────────────────┐
          │ internal/resolve    │   │ internal/registry     │
          │ selection -> plan   │   │ fetch/cache/verify    │
          └─────────┬───────────┘   └─────┬─────────────────┘
                    │                     │
                    │             ┌───────▼───────────────┐
                    │             │ internal/manifest     │
                    │             │ schema + validation   │
                    │             └───────────────────────┘
                    │
          ┌─────────▼───────────┐
          │ internal/render     │   strict text/template
          │ internal/patch      │   anchors + structured patches
          │ internal/engine     │   staging, apply, rollback
          └─────────┬───────────┘
                    │
          ┌─────────▼───────────┐
          │ internal/project    │   state, lockfile, add/remove
          │ internal/fsx        │   atomic writes, path safety
          │ internal/gitx       │   git init
          └─────────────────────┘
```

| Package | Responsibility | Status |
| --- | --- | --- |
| `internal/cli` | Command tree, flags, output formatting, exit behavior | Implemented |
| `internal/tui` | Interactive wizard and Charm-style terminal styling | Implemented |
| `internal/manifest` | Schema types, strict YAML loading, validation | Implemented |
| `internal/registry` | Local/HTTP sources, checksum verification, XDG cache | Implemented |
| `internal/resolve` | Validate selections; build ordered generation plan | Implemented |
| `internal/render` | Strict template rendering, variables, generated secrets | Implemented |
| `internal/patch` | Anchor operations, config patches, dependency adapters | Implemented |
| `internal/engine` | Preflight, staging, atomic commit, mutation transactions | Implemented |
| `internal/project` | `.smokeforge.json`, lockfile, hashes, drift, backups | Implemented |
| `internal/install` | Per-stack, idempotent dependency installation | Planned |
| `internal/version` | Build-time version metadata | Implemented |

## Core principles

### 1. Templates are data

A template consists of files, variables, anchors, and declarative operations.
It cannot execute commands. This makes third-party registries viable later and
keeps generation auditable.

### 2. Validate before writing

Resolution and preflight complete before the first byte is written to the
target. Preflight checks:

- selection compatibility (kinds, limits, variants, `requires`, `conflicts`);
- target directory safety (missing or empty);
- every anchor referenced by an overlay exists exactly once;
- every structured patch path is well-formed and conflict-free;
- output paths are relative and cannot escape the target.

### 3. Atomic by default

Creation renders into a sibling staging directory on the same filesystem, then
renames into place. Mutation (`add`/`remove`) snapshots impacted files to
`.smokeforge/backups/<timestamp>/` and restores them on failure.

### 4. Pinned and verifiable

Registry tarballs are verified with SHA-256 before extraction. Projects pin the
registry tag, resolved commit, and template/overlay checksums in a lockfile.
`--offline` works from cache only.

### 5. Composable, not combinatorial

One base template per stack, plus feature overlays. Instead of maintaining a
preset directory for every feature combination, overlays insert code at named
anchors and patch dependency/config files through language-aware adapters.

## Generation pipeline

```text
1. Parse CLI flags            ──┐
2. Fill gaps with wizard        │ (wizard only when TTY and --no-input absent)
3. Load registry manifest       │ cache first; fetch+verify if stale/missing
4. Resolve and validate         │ compatibility matrix + dependency order
5. Build generation plan        │ targets, templates, overlays, variables, ops
6. Preflight                    │ anchors, patches, target dir, path safety
7. Render into staging          │ strict templates
8. Apply overlays in order      │ verify each operation
9. Write state + lock + .env    │ hashes, registry pin, generated secrets
10. git init                    │ optional, no commit
11. Atomic rename               │ staging -> final target
```

Any failure before step 11 leaves the target untouched. The staging directory is
reported (and optionally retained with a debug flag) for inspection.

## Error and recovery model

| Stage | On failure |
| --- | --- |
| Registry fetch | No target changes; cache untouched until checksum passes |
| Resume/validation | Actionable error listing valid alternatives |
| Preflight | No target changes; all problems reported before writing |
| Staging/apply | Staging removed (or kept for debugging); target untouched |
| Atomic rename | Error returned; staging retained for manual recovery |
| `add`/`remove` | Backup restored automatically; state/lock not advanced |
| `install` | Non-destructive and rerunnable; logs streamed |

## Security posture

- No shell execution from registry content.
- Checksum verification for every download.
- Tar extraction rejects absolute paths, `..` traversal, and links.
- Registry tags are validated before being used in URLs or cache paths.
- Only relative, target-contained output paths are allowed.
- Secrets in `.env` are generated locally with `crypto/rand`.
- No telemetry.

## Related documents

- [Visual architecture with Mermaid diagrams](../architecture.md)
- [CLI and lifecycle](cli.md)
- [Generation engine](generation-engine.md)
- [Project state](project-state.md)
- [Fullstack composition](fullstack.md)
- [Registry manifest](../registry/manifest.md)
- [Overlay model](../registry/overlays.md)
