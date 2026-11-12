# Project state and recovery

**Status:** planned. This document defines the files the generator writes and
how later mutations stay safe.

## Files written into a generated project

| Path | Purpose | Committed to git |
| --- | --- | --- |
| `.smokeforge.json` | Choices, targets, applied overlays, post-generation file hashes | Yes |
| `.smokeforge.lock.json` | Pinned registry tag, resolved commit, template/overlay checksums | Yes |
| `.env` | Local configuration and generated secrets | No (gitignored) |
| `.env.example` | Documented variables with safe placeholders | Yes |
| `.smokeforge/backups/<timestamp>/` | Pre-mutation snapshots for automatic rollback | No (gitignored) |

## `.smokeforge.json`

Tool-managed. Editing by hand is not supported; `doctor` reports invalid state
rather than guessing.

```json
{
  "schemaVersion": 1,
  "name": "acme",
  "kind": "fullstack",
  "createdAt": "2026-11-12T10:15:00Z",
  "registry": {
    "source": "https://github.com/BoatengPeter/smokeforge-registry/releases",
    "tag": "v1.0.0",
    "commit": "9f2c1d4..."
  },
  "variables": {
    "project_name": "acme",
    "project_slug": "acme",
    "module_path": "github.com/acme/acme"
  },
  "targets": [
    {
      "path": "apps/api",
      "stack": "go/chi",
      "features": [
        "database:postgres",
        "cache:redis",
        "queue:rabbitmq",
        "auth:jwt",
        "totp"
      ]
    },
    {
      "path": "apps/web",
      "stack": "next",
      "features": []
    }
  ],
  "rootFeatures": ["docker", "ci:github"],
  "applied": [
    {
      "overlay": "cache:redis",
      "target": "apps/api",
      "registryTag": "v1.0.0",
      "files": {
        "internal/cache/redis.go": "sha256:6b1f...",
        "internal/app/app.go": "sha256:91ab..."
      },
      "markers": ["cache:redis"]
    }
  ]
}
```

Notes:

- `files` records hashes after generation for every file the overlay touched.
- `markers` lists the begin/end marker names inserted, enabling exact removal.
- `rootFeatures` target the repository root rather than a specific app.

## `.smokeforge.lock.json`

```json
{
  "lockVersion": 1,
  "registry": {
    "source": "https://github.com/BoatengPeter/smokeforge-registry/releases",
    "tag": "v1.0.0",
    "commit": "9f2c1d4e8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d",
    "archiveSHA256": "e3b0c44298fc1c149afbf4c8996fb924..."
  },
  "templates": {
    "backend/go/chi": "sha256:...",
    "frontend/next": "sha256:..."
  },
  "overlays": {
    "database/postgres": "sha256:...",
    "cache/redis": "sha256:..."
  }
}
```

The lockfile makes regeneration auditable and lets `add` resolve overlays
against the exact registry content the project was created from.

## Drift detection

Before `add` or `remove` mutates anything:

1. Load state and lock.
2. For every file the operation would touch, compare the current SHA-256 with
   the hash recorded at generation (or after the last successful mutation).
3. Classify: unchanged, modified, deleted, or newly created at an overlapping
   path.
4. If drift exists:
   - show a per-file summary;
   - interactive: ask for confirmation (default: abort);
   - `--no-input`: refuse unless `--force`;
   - `remove`: refuse to delete a modified overlay-added file at all.
5. `doctor` performs the same comparison read-only and reports drift.

Hashes are recorded for generated content only; files created by the user in
generated directories are not managed.

## Mutation transaction

```text
add/remove
  1. resolve overlay(s) against locked registry
  2. validate ops against current files (anchors, patch paths)
  3. snapshot impacted files -> .smokeforge/backups/<timestamp>/
  4. apply atomic per-file writes + structured patches
  5. verify results (re-parse patched files, confirm markers)
  6. update .smokeforge.json + .smokeforge.lock.json
  7. delete snapshot on success (keep on failure / --keep-backups)
```

Recovery guarantees:

- Failure before step 4: nothing changed.
- Failure during step 4/5: snapshot restored automatically; state untouched.
- Failure writing state: files are restored; rerunning is safe and idempotent.
- Process crash: the timestamped backup remains; `doctor` detects a stale
  transaction marker and reports the restore command.

A transaction marker (`.smokeforge/transaction.json`) is written before mutation
and removed after state commits, so interrupted runs are detectable.

## `.env` generation

- Written from the aggregated `.env.example` at creation time.
- Existing `.env` is never overwritten by `add`; new keys are appended.
- Secrets are generated locally with `crypto/rand` and printed only to `.env`.
- `.gitignore` is ensured to contain `.env` and `.smokeforge/`.
- Compose files and app config reference the same variable names.

## Reproducibility

For a pinned registry and identical variables, regeneration yields identical
tracked files. Generated secrets are the only intentional exception. The
lockfile is the source of truth for "what generated this project"; a future
`smokeforge recreate --from-lock` command is planned but out of v1.

## Related documents

- [Generation engine](generation-engine.md)
- [CLI lifecycle](cli.md)
- [Registry manifest](../registry/manifest.md)
```
