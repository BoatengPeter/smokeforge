# Glossary

Domain vocabulary for smokeforge. Use these terms consistently in issues,
specs, code, tests, and docs.

## Core

- **smokeforge** — The Go CLI that generates projects from a template registry.
- **registry** — A versioned repository of templates, overlays, and a manifest,
  fetched by the CLI and pinned by generated projects.
- **manifest** — The registry's top-level `manifest.yaml`, declaring kinds,
  stacks, features, variants, and compatibility rules.
- **stack** — A base template for one language/platform, e.g. `next`, `go/chi`.
- **kind** — A target category: backend, frontend, mobile, desktop.
- **base template** — The complete starter tree for one stack, before overlays.
- **feature** — A composable capability such as database, cache, queues, auth,
  TOTP, analytics, Docker, or CI.
- **variant** — A selectable implementation of a feature, e.g. `postgres`,
  `redis`, `jwt`, `session`, `posthog`.
- **overlay** — A declarative set of operations that adds one feature variant to
  a base template. Overlays are data; they never execute code.
- **template bundle** — A named selection of stack + features used as a preset.
  Not implemented yet.
- **target** — One generated application directory. A fullstack project will
  have multiple targets under `apps/`.
- **standalone** — A project with no backend feature overlays.
- **fullstack** — A project combining a frontend and a backend, or a Next app
  with backend overlays.

## Generation

- **selection** — The user's kind, stack, features, variables, and project name.
- **plan** — The validated, ordered generation output: base template, overlays,
  targets, and variables.
- **resolver** — The component that validates a selection and builds a plan.
- **renderer** — Strict `text/template` rendering for files and paths.
- **anchor** — A named comment in a base file where overlays insert code.
- **marker** — Begin/end comments wrapping an inserted block so it can be
  removed exactly.
- **patch operation** — A declarative overlay operation: add files, insert at an
  anchor, or patch JSON/YAML/TOML/dependencies/env.
- **staging** — A temporary generation directory; the target is committed by
  atomic rename only after success.
- **lockfile** — `.smokeforge.lock.json`, pinning registry tag/commit and
  template/overlay checksums.
- **project state** — `.smokeforge.json`, recording choices, targets, applied
  overlays, env keys, and per-file hashes.
- **drift** — A tracked file whose current hash differs from project state.
- **backup** — A timestamped snapshot under `.smokeforge/backups/` taken before
  a mutation.
- **rollback** — Restoring the backup when a mutation fails.
- **lifecycle** — The `add`, `remove`, `install`, and `doctor` operations after
  creation.

## Next.js template

- **Route Handler** — The Next backend API implemented in
  `src/app/api/**/route.ts`.
- **RSC boundary** — The server/client component boundary; keep serialized
  props minimal.
- **per-request store** — A Zustand store created for one request/render tree,
  never shared at module scope.
- **consent-gated analytics** — PostHog capture only after explicit opt-in.
