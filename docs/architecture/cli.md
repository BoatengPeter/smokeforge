# CLI and lifecycle

**Status:** `new`, `version`, `list`, `registry info/update`, and `doctor` are
implemented. `add`, `remove`, `install`, `registry pin`, fullstack generation,
and `--json` output are planned.

## Command tree

```text
smokeforge
├── new [dir]                 scaffold a project (implemented)
├── add <feature>...          apply overlays later (planned)
├── remove <feature>...       revert overlays (planned)
├── install                   install target dependencies (planned)
├── list [stacks|features]    inspect registry contents (implemented)
├── registry
│   ├── info                  show source, tag, schema, counts (implemented)
│   ├── update                refresh the cached registry (implemented)
│   └── pin <tag>             pin a registry tag (planned)
├── doctor                    check environment and registry health (implemented)
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
4. **Features** — filtered by the chosen stack. Variant features appear as
   `feature — variant`; boolean features appear by name. Requirement and
   conflict errors are reported before generation.
5. **Variables** — one prompt per template variable, prefilled with the flagged
   value or registry default.
6. **Review and generate** — the CLI prints a project summary and asks for
   confirmation, then renders staged progress and a generated-project box.

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

Flags: `--kind`, `--stack`, `--name`, `--feature id[:variant]` (repeatable),
`--var key=value` (repeatable), `--no-input`, `--no-git`, `--dry-run` (render and
validate without writing), and `--keep-staging` (retain staging for debugging).
Fullstack flags (`--clients`, `--backend`) and `--json` are planned.

### Variable precedence

1. Explicit `--var` values.
2. Wizard answers (which are prefilled from `--var` values).
3. Template defaults from `template.yaml`.

`project_name` and `project_slug` are provided automatically. Unknown variables
are errors. Missing values in `--no-input` mode are errors, never silently
defaulted; a declared non-empty default satisfies the variable.

## `smokeforge add` (planned)

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

## `smokeforge remove` (planned)

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
3. `git` availability (warning only, since `--no-git` exists).
4. Project state integrity when run inside a generated project: schema version,
   recorded targets exist, hashes match (reports drift without changing files).
5. Toolchains for recorded targets (informational; never fails the command).

Exit code is non-zero only if a fatal check fails.

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
