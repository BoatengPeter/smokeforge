# CLI and lifecycle

**Status:** `new`, `add`, `remove`, `version`, `list`, `registry info/update`,
`doctor`, and shell completion are implemented. `install`, `registry pin`,
mutation dry-runs, and `--json` output are planned.

## Command tree

```text
smokeforge
├── new [dir]                 scaffold a project (implemented)
├── add <feature>...          add reversible overlays (implemented)
├── remove <feature>...       revert reversible overlays (implemented)
├── install                   install target dependencies (planned)
├── list [stacks|features]    inspect registry contents (implemented)
├── registry
│   ├── info                  show source, tag, schema, counts (implemented)
│   ├── update                refresh the cached registry (implemented)
│   └── pin <tag>             pin a registry tag (planned)
├── doctor                    check environment, registry, project drift (implemented)
├── completion <shell>        generate bash/zsh/fish/PowerShell completion
└── version                   print version information (implemented)
```

## Global flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `--registry <url\|dir>` | built-in registry URL | Use a custom registry. A local directory is loaded directly. |
| `--tag <tag>` | `latest` | Registry release tag. Validated before use. |
| `--cache-dir <dir>` | user cache directory | Override the registry cache location. |
| `--offline` | `false` | Forbid network; require cache. Fails clearly if missing. |

`NO_COLOR` disables terminal color. `--no-input` is a `new` flag, not a global
flag. Third-party registry environment overrides and `--json` are planned.

## `smokeforge new`

```text
smokeforge new [dir] [flags]
```

If `dir` is omitted, it defaults to the slug of the project name. The target must
not exist or must be empty. M2 supports single-target kinds; fullstack and
multi-client selection are planned for M4.

The command prints a boxed summary, asks for confirmation in a TTY, and reports
successful stages with checkmarks. `--no-input` produces the same output without
prompts, and `NO_COLOR=1` disables color.

### Wizard flow

1. **Kind** — every registry kind whose `max` is at least 1 and that has a
   stack.
2. **Stack** — filtered to the chosen kind; one stack for M2.
3. **Name** — the display name; `project_slug` is derived from it.
4. **Preset** — when the registry declares presets, choose a starting bundle.
   Presets may declare kinds; the default preset for the selected kind is
   preselected.
5. **Features** — filtered by the chosen stack, prefilled from the preset.
   Variant features appear as `feature — variant`; boolean features appear by
   name. Required companions are added automatically.
6. **Variables** — one prompt per template variable, prefilled with the flagged
   value or registry default.
7. **Review and generate** — the CLI prints a project summary and asks for
   confirmation, then renders staged progress and a generated-project box.

The wizard runs as one grouped form. `Shift+Tab` moves back to the previous
step so earlier answers can be changed without aborting.

### Flag parity

Every wizard choice has a flag; CI never needs a TTY.

```bash
smokeforge new ./acme \
  --registry ./testdata/registry \
  --kind backend \
  --stack test/plain \
  --name "Acme App" \
  --feature logging \
  --var module_path=example.com/acme \
  --no-input --no-git
```

Flags: `--kind`, `--stack`, `--name`, `--preset`, `--feature id[:variant]`
(repeatable), `--var key=value` (repeatable), `--no-input`, `--no-git`,
`--dry-run` (render and validate without writing), and `--keep-staging` (retain
staging for debugging). Fullstack flags (`--clients`, `--backend`) and `--json`
are planned.

With `--no-input`, an empty feature selection uses the registry's default
preset for the selected kind. Explicit features are merged with an explicit
preset, and requirements are resolved transitively. Selecting a preset outside
its declared kinds is an error.

### Variable precedence

1. Explicit `--var` values.
2. Wizard answers (which are prefilled from `--var` values).
3. Template defaults from `template.yaml`.

`project_name` and `project_slug` are provided automatically. Unknown variables
are errors. Missing values in `--no-input` mode are errors, never silently
defaulted; a declared non-empty default satisfies the variable.

## `smokeforge add`

```text
smokeforge add <feature[:variant]>... [--project-dir <dir>] [--force]
```

- Locates the project by walking up from `--project-dir` (default `.`) to
  `.smokeforge.json`.
- Loads the registry from the project's recorded source and locked tag; pass
  `--registry` to override the source for development.
- Resolves the combined feature selection, auto-adds required companions, and
  applies only the new overlays in dependency order.
- Refuses when any tracked file drifted from its recorded SHA-256. `--force`
  overrides this for adds; the user's edits are preserved.
- Backs up every impacted file plus state/lockfiles, applies the overlays,
  updates env files, rehashes tracked files, and writes state/lock.
- On any failure, restores the backup and leaves state unchanged.
- Prints a boxed summary with the added overlays and resulting feature list.

## `smokeforge remove`

```text
smokeforge remove <feature[:variant]>... [--project-dir <dir>] [--cascade]
```

- Locates the project and locked registry like `add`.
- Rejects features that are not applied.
- Rejects removal when an active feature requires the target. Pass `--cascade`
  to remove dependents first.
- Refuses when tracked files drifted. Removal is always strict; there is no
  force flag.
- Automatically reverses only:
  - files added by overlays (deleted),
  - marked insertion blocks (excised),
  - env keys contributed by the overlay (removed from `.env` and `.env.example`).
- Refuses overlays containing `patchJSON`, `patchYAML`, `patchTOML`, or
  `addDependency` with a manual-removal message.
- Backs up all impacted files, removes in reverse application order, rehashes
  remaining overlays, updates state/lock, and restores on failure.

## `smokeforge install` (planned)

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
smokeforge list [stacks|features]
```

Prints a tab-aligned table from the validated registry manifest. `--json` output
is planned.

## `smokeforge registry`

| Subcommand | Behavior | Status |
| --- | --- | --- |
| `info` | Source, resolved tag, schema version, counts. | Implemented |
| `update` | Re-downloads and verifies the selected tag; replaces cache atomically. Refuses in `--offline` mode. | Implemented |
| `pin <tag>` | Updates a project lockfile to a new tag. | Planned |

## `smokeforge doctor`

Checks, in order:

1. Registry load/validation (cache or network, per flags).
2. Cache directory writability.
3. `git` availability.
4. When run inside a generated project: state/lock validity and drift against
   recorded hashes. Drift is reported with the affected paths and fails the
   command. Toolchain checks are planned.

Exit code is non-zero only if a fatal check fails.

## Help and completion

Every command supports `--help`, and `smokeforge help <command>` is equivalent:

```bash
smokeforge --help
smokeforge help new
smokeforge new --help
smokeforge remove --help
```

Help text includes purpose, examples, flags, and required-companion behavior. It
does not load the registry or access the network.

Generate completion scripts with:

```bash
smokeforge completion bash
smokeforge completion zsh
smokeforge completion fish
smokeforge completion powershell
```

The scripts are written to stdout.

## Output and exit conventions

- Human output: aligned tables, Charm-style summaries, checkmark stages.
- Color is disabled automatically for non-TTY output and by `NO_COLOR`.
- `--json`: single-document JSON, stable field names. (Planned.)
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
