# CLI and lifecycle

**Status:** command surface and semantics are designed; only `version` is
implemented, with `list`/`registry`/`doctor` landing in the foundation
milestone. `new`, `add`, `remove`, and `install` are planned.

## Command tree

```text
smokeforge
├── new [dir]                 scaffold a project
├── add <feature>...          apply feature overlays to an existing project
├── remove <feature>...       revert feature overlays
├── install                   install dependencies for recorded targets
├── list [stacks|features]    inspect registry contents
├── registry
│   ├── info                  show source, tag, schema, counts
│   ├── update                refresh the cached registry
│   └── pin <tag>             pin a registry tag for the current project
├── doctor                    check environment, cache, registry, toolchains
└── version                   print version information
```

## Global flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `--registry <url\|dir>` | built-in registry URL | Use a custom registry. A local directory is loaded directly. |
| `--tag <tag>` | `latest` | Registry release tag. Validated before use. |
| `--offline` | `false` | Forbid network; require cache. Fails clearly if missing. |
| `--json` | `false` | Machine-readable output for supported commands. |
| `--no-input` | `false` | Never prompt; missing required choices are errors. |

Environment overrides: `SMOKEFORGE_REGISTRY`, `SMOKEFORGE_REGISTRY_TAG`,
`NO_COLOR`.

## `smokeforge new`

```text
smokeforge new [dir] [flags]
```

If `dir` is omitted, the wizard asks for it. The target must not exist or must be
empty.

### Wizard flow

1. **Kind** — backend, frontend, mobile, desktop, or fullstack.
2. **Stack(s)** — filtered to valid choices for the kind:
   - backend: `go/chi`, `rust/axum`, `python/fastapi`
   - frontend: `next`
   - mobile: `ios/swiftui`, `android/compose` (multi-select)
   - desktop: `desktop/gpui`, `desktop/egui` (single select)
   - fullstack: one API stack + any supported clients
3. **Features** — filtered by the selected stacks:
   - `database:postgres`, `cache:redis`, `queue:rabbitmq`, `auth:jwt`,
     `totp`, `docker`, `ci:github`
   - dependent features are disabled until their requirements are selected
     (`totp` requires `auth:jwt`)
4. **Details** — project name, module path / package / bundle identifiers,
   ports. Defaults are stack-aware; collisions are rejected.
5. **Review** — target tree preview, file counts, registry tag, operation
   summary. Confirmation is required.
6. **Generate** — progress output; on failure the target is untouched and the
   staging path is reported.

### Flag parity

Every wizard step has a flag; CI never needs a TTY.

```bash
smokeforge new ./acme \
  --kind fullstack \
  --backend go/chi \
  --clients next,ios/swiftui \
  --feature database:postgres \
  --feature cache:redis \
  --feature queue:rabbitmq \
  --feature auth:jwt \
  --feature totp \
  --feature docker \
  --feature ci:github \
  --module github.com/acme/acme \
  --name acme \
  --no-input
```

Additional flags: `--dry-run` (print the resolved plan and operations, write
nothing), `--no-git`, `--keep-staging`, `--json`.

### Variable precedence

1. Explicit flags.
2. Answers from the wizard.
3. Stack defaults from the registry.
4. Built-in defaults (ports, package names).

Missing values in `--no-input` mode are errors, never silently defaulted, unless
the stack declares an explicit default.

## `smokeforge add`

```text
smokeforge add <feature[:variant]>... [--dry-run] [--yes]
```

- Must run inside a generated project (locates `.smokeforge.json` upward).
- Resolves overlays against the project's locked registry version by default.
  `--registry-tag <tag>` opts into a newer registry explicitly.
- Applies only the requested overlays to their recorded targets.
- Compares every touched file's current SHA-256 against the hash recorded after
  generation. Drifted files are never silently overwritten: the command shows a
  summary and asks for confirmation; `--force` overrides, `--no-input` refuses.
- Prompts for any new variables the overlay needs.
- Updates `.smokeforge.json` and the lockfile only after successful commit.

## `smokeforge remove`

```text
smokeforge remove <feature[:variant]>... [--dry-run] [--yes]
```

Removal is conservative and reversible:

- Deletes files added by the overlay only if their hashes still match.
- Excises inserted blocks using `smokeforge:begin/end` markers.
- Reverses structured patches only when the affected entries are unchanged.
- Refuses non-removable overlays (region replacements) with an explanation and
  a manual-removal hint.
- Restores backups automatically if any step fails.

## `smokeforge install`

A deliberately separate step, so project creation never blocks on toolchains or
network package registries.

```text
smokeforge install [--yes] [--target <path>]
```

- Reads recorded targets from `.smokeforge.json`.
- Runs each stack's standard, idempotent setup commands, streaming output:
  - Go: dependency download/tidy
  - Rust: `cargo fetch`
  - Python: `uv sync` (or `pip install -e .` fallback)
  - Node: package manager install based on lockfile
  - Android: Gradle sync/build check
  - iOS: dependency resolution / `swift build` where applicable
- Never executes strings from templates. Commands come from built-in adapters.
- Failures are non-destructive and safe to rerun.

## `smokeforge list`

```text
smokeforge list [stacks|features] [--json]
```

Prints a tab-aligned table. With `--json`, emits the validated registry manifest
subset for the requested category.

## `smokeforge registry`

| Subcommand | Behavior |
| --- | --- |
| `info` | Source, resolved tag, cache location, schema version, counts. |
| `update` | Re-downloads and verifies the selected tag; replaces cache atomically. Refuses in `--offline` mode. |
| `pin <tag>` | Updates the project's lockfile to a new tag after validating compatibility; does not regenerate files. |

## `smokeforge doctor`

Checks, in order:

1. Registry load/validation (cache or network, per flags).
2. Cache directory writability.
3. `git` availability (warning only, since `--no-git` exists).
4. Project state integrity when run inside a generated project: schema version,
   recorded targets exist, hashes match (reports drift without changing files).
5. Toolchains for recorded targets (informational; never fails the command).

Exit code is non-zero only if a fatal check fails.

## Output and exit conventions

- Human output: plain text, aligned tables, no decorative noise.
- `--json`: newline-delimited or single-document JSON, stable field names.
- Exit codes:
  - `0` success (including warnings)
  - `1` operational failure (registry, validation, generation)
  - `2` usage error (unknown command/flag, invalid arguments)
- Errors are actionable: state what failed, why, and the valid alternatives.

## Related documents

- [Generation engine](generation-engine.md)
- [Project state](project-state.md)
- [Registry manifest](../registry/manifest.md)
```
