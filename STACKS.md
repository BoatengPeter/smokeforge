# smokeforge stack catalog

The stacks, versions, and defaults smokeforge targets. M3 is in progress; the
Next.js slice is implemented first. Items marked **deferred** are not yet shipped
and require the decisions noted.

## Status legend

| Status | Meaning |
| --- | --- |
| Implemented | Generated and validated end-to-end |
| In progress | Actively being built in the registry |
| Provisional | Language/platform chosen; package-level choices pending |
| Deferred | Explicitly out of the current milestone |

## CLI and tooling

| Component | Version | Notes |
| --- | --- | --- |
| Go | 1.26.11+ | Security-patched toolchain |
| Cobra | 1.10.2 | Command tree |
| huh | 1.0.0 | Interactive wizard |
| lipgloss | 1.1.0 | Terminal styling |
| yaml.v3 | 3.0.1 | Manifest/overlay parsing |
| go-toml/v2 | 2.4.3 | TOML patches |
| x/mod | 0.43.0 | `go.mod` dependency edits |
| Lint | golangci-lint v2.11.0 | 48 linters including gosec |
| Security | govulncheck | CI on every push/PR |
| Release | goreleaser v2 | Linux/macOS/Windows amd64/arm64 |

CLI tests: Go unit/integration/race, generated fixture build, cross-build matrix.

## Frontend — Next.js

**Status:** base and core overlays implemented; standalone, fullstack auth, Docker, CI, and analytics wired.

| Component | Version | Notes |
| --- | --- | --- |
| Next.js | 16.4.6 | App Router, Turbopack |
| React | 19.3.0 | Matches official `create-next-app` |
| TypeScript | 5.9.3 | Pinned; TS 7 upgrade after registry CI |
| Node.js | 24 LTS | `.nvmrc` 24 |
| Package managers | pnpm 11.13.1 default; npm 11.18.0; bun 1.3.14 | Chosen via `package_manager` |
| Tailwind CSS | 4.3.3 | `@tailwindcss/turbopack` 4.3.3 |
| UI system | shadcn/ui (CLI-generated) | Lucide icons |
| Motion | 14.13.0 | Scroll reveals, reduced-motion support, interaction feedback |
| Logging | Pino 10.4.0 | Server-side structured logs |
| ESLint | 9.39.5 | Flat config + `eslint-config-next` 16.4.6 |
| Jest | 30.5.2 | `next/jest`, jsdom |
| Testing Library | React 16.3.3; jest-dom 7.0.1 | Component tests |
| Playwright | 1.65.0 | E2E overlay |
| React Hook Form | 7.92.0 | Auth/forms overlay; `@hookform/resolvers` 7.0.6 |
| Zustand | 5.1.0 | State overlay; per-request provider for SSR safety |
| Zod | 4.6.7 | API and form validation |
| Postgres | 17 | Optional database overlay |
| Drizzle ORM | 0.45.4 | `drizzle-kit` 0.31.11 |
| Better Auth | 1.7.22 | Email/password + email OTP sessions |
| PostHog JS | 1.447.15 | Optional privacy-first overlay |
| Deployment | Node/Docker | `output: "standalone"` |

### Defaults

- Master preset is `fullstack`: `database:postgres`, `auth:session`, `state:zustand`.
- `frontend` is the base-only preset; `production` adds E2E, Docker, CI, and PostHog.
- Required companions are auto-added; removal of a required feature is blocked
  unless `--cascade` is explicit.
- One composable `next` base; standalone = base only, fullstack = base + overlays.
- Backend is standard `src/app/api/**/route.ts` Route Handlers.
  - `/api/health`, `/api/auth/[...all]`, versioned resources under `/api/v1`.
  - Success `{ "data": ... }`; errors `{ "error": { "code", "message", "fields" } }`.
- App Router with `src/`, `@/*` alias, Server Components by default.
- Metadata API, generated OpenGraph image, sitemap, robots.
- Tailwind v4 CSS-first tokens; no `tailwind.config.ts`.
- Forms: React Hook Form + Zod with auth. State: Zustand provider with the state
  overlay; no module-level request state.
- Jest for unit tests; Playwright for e2e with the testing overlay.
- PostHog: client-only, opt-in, autocapture off, no session recording.
- Lockfiles are created by the first install (pnpm-lock.yaml, package-lock.json,
  or bun.lock). Exact dependency pins ship in `package.json`.

### Project-scoped skills

Generated Next apps receive these under `.agents/skills` and `.claude/skills`,
plus `AGENTS.md`, `CLAUDE.md`, and `skills-lock.json`:

- `nextjs-app-router-patterns`
- `better-auth-best-practices`
- `better-auth-security-best-practices`
- `github-actions-templates`
- `zustand`
- `vercel-react-best-practices`
- `vercel-composition-patterns`
- `frontend-design`
- `web-design-guidelines`
- `tailwind-design-system`
- `zod`
- `robots-txt`

They are scoped to the generated frontend app, not the smokeforge CLI repo.

### Planned overlays

| Overlay | Status | Notes |
| --- | --- | --- |
| `database:postgres` | Implemented | Drizzle schema, migrations, compose override |
| `state:zustand` | Implemented | Store, actions, provider, selectors, tests |
| `testing:e2e` | Implemented | Playwright config, smoke test, Makefile targets |
| `auth:session` | Implemented | Better Auth email/password, OTP verification/reset |
| `totp` | Deferred | Better Auth TOTP plugin later |
| `analytics:posthog` | Implemented | Client-only, consent-first |
| `docker` | Implemented | Non-root multi-stage, package-manager aware |
| `ci:github` | Implemented | GitHub Actions; requires `testing:e2e` |
| `cache:redis` | Deferred | After deployment/DB flow is proven |
| `queue` | Deferred | Next worker model unsupported in this pass |

## Backend options — provisional

Language/platform choices are recorded, but HTTP frameworks, ORMs, clients, and
migration tools still require discussion.

| Stack ID (provisional) | Language | Framework | Status |
| --- | --- | --- | --- |
| `go/chi` | Go | chi candidate | Provisional |
| `rust/axum` | Rust | axum candidate | Provisional |
| `python/fastapi` | Python | FastAPI candidate | Provisional |

Database, cache, queue, auth, and TOTP implementation packages for these stacks
remain undecided.

## Mobile — provisional

| Stack ID | Platform | UI | Status |
| --- | --- | --- | --- |
| `ios/swiftui` | Swift/iOS | SwiftUI candidate | Provisional |
| `android/compose` | Kotlin/Android | Jetpack Compose candidate | Provisional |

## Desktop — provisional

| Stack ID | Framework | Status |
| --- | --- | --- |
| `desktop/gpui` | GPUI | Under consideration |
| `desktop/egui` | EGUI | Under consideration |

## Infrastructure

| Area | Default | Status |
| --- | --- | --- |
| Containers | Docker, multi-stage, non-root | In progress for Next |
| CI | GitHub Actions | In progress for Next |
| Database | Postgres 17 | In progress for Next |
| Cache | Redis | Deferred for Next |
| Queue | RabbitMQ (backend stacks) | Deferred for Next |
| Storage/search | MinIO/imgproxy/nginx, R2, Meilisearch/Elasticsearch | Deferred |
| Registry | GitHub release tarball + SHA-256 | Implemented |

## Registry format

- Single registry repository with a manifest, templates, overlays, and tests.
- Releases publish `registry.tar.gz` (top-level `registry/`) and `registry.sha256`.
- Projects pin registry tag/commit and template/overlay checksums.
- Templates and overlays are declarative data; no executable hooks.

## Platform support

Linux, macOS, and Windows on amd64/arm64. CI tests on all three operating systems,
cross-compiles all six release targets, and runs `govulncheck`.

## Related documentation

- [README](README.md)
- [Registry manifest](docs/registry/manifest.md)
- [Overlay model](docs/registry/overlays.md)
- [Cross-platform compatibility](docs/development/cross-platform.md)
- [Security](SECURITY.md)
- [Roadmap](docs/development/roadmap.md)
- Registry repository: https://github.com/BoatengPeter/smokeforge-registry
