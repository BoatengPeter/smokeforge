# Project state and recovery

**Status:** implemented for creation and mutation (`add`/`remove`). Drift
detection and backup/restore are enforced for lifecycle commands.

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
    "commit": ""
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
        "internal/app/app.go": "sha256:91ab...",
        ".env.example": "sha256:48cd..."
      },
      "markers": ["cache:redis"],
      "env": ["REDIS_URL"]
    }
  ]
}
```

Notes:

- `commit` and `archiveSHA256` are currently empty for remote/local registries;
  populated remote archive digests arrive with a later registry metadata change.
- `files` records hashes after generation for every file the overlay touched,
  including `.env.example` when the overlay contributes env keys. `.env` is not
  hashed because it contains local secrets.
- `markers` lists the begin/end marker names inserted, enabling exact removal.
- `env` lists environment keys contributed by the overlay.
- `rootFeatures` target the repository root rather than a specific app.

## `.smokeforge.lock.json`

```json
{
  "lockVersion": 1,
  "registry": {
    "source": "https://github.com/BoatengPeter/smokeforge-registry/releases",
    "tag": "v1.0.0",
    "commit": "",
    "archiveSHA256": ""
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

Before `add` or `remove` mutates anything (implemented):

1. Load state and lock.
2. For every file the operation would touch, compare the current SHA-256 with
   the hash recorded at generation (or after the last successful mutation).
3. Classify: unchanged, modified, deleted, or newly created at an overlapping
   path.
4. If drift exists:
   - `add`: refuse and report the paths; `--force` proceeds and preserves edits;
   - `remove`: always refuse;
   - `doctor`: reports drift read-only and fails.
5. `.env` is intentionally excluded from drift because it holds local secrets.

Hashes are recorded for generated content only; files created by the user in
generated directories are not managed.

## Mutation transaction

```text
add/remove
  1. resolve overlay(s) against the locked registry
  2. refuse on drift (add: --force may override)
  3. snapshot impacted files + state/lock -> .smokeforge/backups/<timestamp>/
  4. apply or revert overlays; update env files
  5. rehash every tracked file for all applied overlays
  6. update .smokeforge.json + .smokeforge.lock.json
  7. discard snapshot on success
```

Recovery guarantees (implemented):

- Failure before step 3: nothing changed.
- Failure during steps 4-6: snapshot restored automatically; state untouched.
- A failed restore returns both errors and reports the retained backup path.
- Backups of files that did not exist at snapshot time cause restore to delete
  any partially created file.
- A process crash mid-mutation can leave a timestamped backup; automatic crash
  recovery markers are not implemented yet.

## `.env` generation

- Written from the aggregated `.env.example` at creation time.
- Existing `.env` is never overwritten by `add`; new keys are appended.
- Secrets are generated locally with `crypto/rand` and printed only to `.env`.
- `.gitignore` is ensured to contain `.env` and `.smokeforge/`.
- Compose files and app config reference the same variable names.

## Reproducibility

For a pinned registry and identical variables, generation yields identical
tracked files. Generated secrets and `createdAt` are the intentional exceptions. The
lockfile is the source of truth for "what generated this project"; a future
`smokeforge recreate --from-lock` command is planned but out of v1.

## Related documents

- [Generation engine](generation-engine.md)
- [CLI lifecycle](cli.md)
- [Registry manifest](../registry/manifest.md)
```
