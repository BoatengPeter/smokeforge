# Fullstack composition

**Status:** API + Next.js is implemented when `--frontend next` is explicit.
`--mobile` is repeatable for Android and iOS. API + mobile without web uses
`workspace/mobile`; web + mobile uses `workspace/fullstack` plus mobile overlays.
`--desktop electron` adds an Electron target with offline Library
synchronization: local outbox, idempotent mutations, version checks, cursors,
retries, and manual conflicts.

## Kinds and limits

The registry manifest declares per-kind selection limits:

| Kind | Max in a project | Examples |
| --- | --- | --- |
| `backend` | 1 | `go/gin-layered`, `rust/axum` |
| `frontend` | 1 | `next` |
| `mobile` | 2 | `ios/swiftui` + `android/compose` |
| `desktop` | 1 | `desktop/electron`; GPUI/EGUI deferred |

Fullstack generates web only when `--frontend next` is explicit. At least one
client (web, mobile, or desktop) is required.

A fullstack project is one API target plus at least one supported client. API +
mobile without web uses `workspace/mobile`; web + API uses `workspace/fullstack`.
Fullstack generates Next.js only when `--frontend next` is explicit. The
`workspace/mobile` root also supports one or both `apps/android` and `apps/ios`.

## Monorepo layout

```text
acme/
  apps/
    api/                  # backend target (required for API-backed compositions)
    web/                  # Next.js target (explicit --frontend next)
    ios/                  # SwiftUI target
    android/              # Android Compose target
    desktop/              # Electron target (requires explicit web)
  .github/
    workflows/
      ci.yml
  CONTEXT-MAP.md          # API/web/workspace context routing
  compose.yaml            # root compose for web fullstack + docker
  .smokeforge.json
  .smokeforge.lock.json
  README.md
  Makefile                # root entry point; delegates API, web, and mobile targets
```

Target paths are fixed by kind (`apps/api`, `apps/web`, `apps/ios`,
`apps/android`, `apps/desktop`) unless a stack declares an override.

## Composition rules

1. **Root workspace base** — a `workspace/fullstack` template renders the root
   files (README, task runner, `.gitignore`, root env documentation).
2. **One base template per target** — each target is rendered independently
   with its own variables (module path, bundle ID, application ID).
3. **Overlay targeting:**
   - backend features (`database`, `cache`, `queue`, `auth`, `totp`) apply only
     to the API target;
   - frontend features (`data:tanstack-query`, `state`, `analytics`) apply only
     to the web target;
   - `auth:jwt` on the API adds `auth:bff` to Next.js automatically;
   - `docker` applies to both apps and the root; `ci` applies at the root;
   - mobile, desktop, and client-specific overlays apply to their targets.
4. **Operation ordering** — target bases first, then target overlays, then the
   root base and root overlays. Root operations run last so they can aggregate
   generated services and workflows.
5. **No cross-target file writes** — an overlay may not write outside its
   declared target; root overlays use root-relative paths only.

## API-client wiring

Clients must not require manual URL hunting. At generation time:

- The API port is allocated (collision-checked) and recorded in state.
- Web: `.env.example`/`.env` receives server-only `API_URL=http://localhost:<port>`;
  the browser uses same-origin BFF routes.
- iOS: `project.yml` writes `API_BASE_URL` from the `ios_api_base_url` variable; the app reads it from `Info.plist`.
- Android: `BuildConfig.API_BASE_URL`; default `http://10.0.2.2:8080`.
- Desktop: `DESKTOP_API_URL` in the main process; tokens stay in main. SQLite is
  local data/cache, not a second server database.
- Root `.env.example` documents shared values; compose uses the same names.

This wiring is implemented by small client-specific overlays shipped with each
client base, not by hardcoded CLI logic.

## Docker and CI aggregation

`docker` applies the per-target Dockerfiles, then renders a root `compose.yaml`
that extends the API/web compose files, adds Postgres/migrations for the
fullstack preset, and wires `API_URL=http://api:8080`. The root Makefile runs
`docker compose up --build`. `compose config` is part of verification.

`ci` renders `.github/workflows/ci.yml` at the root. It detects Go or Rust,
installs the corresponding toolchain, runs standard API checks, then runs the
web install/typecheck/lint/test/build path and conditional Android/iOS jobs.

Overlays patch compose/workflow files through structured YAML operations, not
string concatenation, so multiple features compose deterministically.

## Adding clients later

`smokeforge add next` on a backend-only project is **not** just an overlay: it
introduces a new target. Planned behavior for v1.1:

1. Detect that the feature is actually a stack/kind addition.
2. Resolve the new base + client wiring overlay.
3. Render the new target under `apps/<kind>`.
4. Apply root updates (compose, CI, env docs).
5. Record the new target in state.

M4 lifecycle supports adding and removing feature overlays on single-target
projects only. Adding client targets and multi-target mutation remain planned.

## Validation

- Fullstack requires an API stack; clients alone use their own kind.
- Fullstack Docker requires the API database overlay because the root compose
  wires migrations and Postgres.
- Multiple clients of the same kind beyond the limit are rejected.
- Desktop GPUI and EGUI are mutually exclusive per project.
- Client/API compatibility is declared in the manifest (`stacks`/`kinds`), not
  assumed by the CLI.

## Related documents

- [CLI lifecycle](cli.md)
- [Registry manifest](../registry/manifest.md)
- [Overlay model](../registry/overlays.md)
```
