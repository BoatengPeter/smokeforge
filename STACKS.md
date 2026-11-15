# smokeforge stack catalog

The stacks, versions, and defaults smokeforge targets. Next.js and the Go/Gin
backend and the Rust/Axum backend are implemented; Python, mobile, and desktop remain provisional.
Items marked **deferred** are not yet shipped and require the decisions noted.

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
| Zustand | 5.1.0 | Client UI state; per-request provider for SSR safety |
| TanStack Query | 5.107.1 | Server-state cache, query keys, and mutation hooks |
| Zod | 4.6.7 | API and form validation |
| Postgres | 17 | Optional database overlay |
| Drizzle ORM | 0.45.4 | `drizzle-kit` 0.31.11 |
| Better Auth | 1.7.22 | Email/password + email OTP sessions |
| PostHog JS | 1.447.15 | Optional privacy-first overlay |
| Deployment | Node/Docker | `output: "standalone"` |

### Defaults

- Master preset is `fullstack`: `database:postgres`, `auth:session`, `state:zustand`, `data:tanstack-query`.
- `frontend` is the base-only preset; add E2E, Docker, CI, and PostHog with feature flags.
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
  overlay; TanStack Query owns server state; no module-level request state.
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
| `data:tanstack-query` | Implemented | QueryClient provider, typed query/mutation/auth hooks |
| `testing:e2e` | Implemented | Playwright config, smoke test, Makefile targets |
| `auth:session` | Implemented | Better Auth email/password, OTP verification/reset |
| `totp` | Deferred | Better Auth TOTP plugin later |
| `analytics:posthog` | Implemented | Client-only, consent-first |
| `docker` | Implemented | Non-root multi-stage, package-manager aware |
| `ci:github` | Implemented | GitHub Actions; E2E steps run only when Playwright exists |
| `cache:redis` | Deferred | Go cache overlay exists; Next.js cache is not planned |
| `queue` | Deferred | Go RabbitMQ overlay exists; Next.js queue is not planned |

## Backend — Go/Gin

**Status:** implemented; all four architecture bases and the PostgreSQL, JWT,
TOTP, Redis cache, RabbitMQ queue/scheduler, leader election, dedupe,
compression, image media, Meilisearch search, queue-backed search sync, signed
pagination, Docker, and CI overlays are wired.

| Component | Version | Notes |
| --- | --- | --- |
| Go | 1.26.11 | Same toolchain baseline as the CLI |
| Gin | 1.12.0 | HTTP router and middleware |
| OpenAPI | `oapi-codegen` 2.5.0 | Spec-first generated Gin server interfaces |
| Swagger UI | `swaggest/swgui` 1.8.9 | Embedded UI served at `/docs` |
| Validation | `go-playground/validator` 10.30.6 | Request boundary validation |
| Config | `caarlos0/env` 11.4.2 | Typed env config; local `godotenv` |
| Logging | Zap 1.28.0 | Structured JSON/console logs |
| Rate limiting | `golang.org/x/time` 0.16.0 | Bounded per-IP token buckets |
| UUID | `google/uuid` 1.6.0 | UUIDv7 identifiers |
| Postgres | 17 | Optional database overlay |
| pgx | 5.13.0 | Connection pool and SQL driver |
| sqlc | 1.31.1 | Compile-checked queries |
| goose | 3.30.1 | Embedded migrations |
| AWS SDK | v1.47.3 | S3-compatible media storage and presigning |
| MinIO | latest | Optional private media bucket |
| imgproxy | 3.30.1 | Optional signed image transformations |
| nginx | 1.28.2 | Optional transform response cache |
| Meilisearch | 1.6 | Optional public full-text search |
| Redis | 8 | Optional cache, dedupe markers, scheduler lease |
| RabbitMQ | 4.1 | Optional durable queue, retries, and DLQ |
| JWT | `golang-jwt/jwt/v5` 5.3.1 | HS256 access and MFA tokens |
| Passwords | `x/crypto` 0.59.0 | Argon2id |
| TOTP | `pquerna/otp` 1.5.0 | Enrollment, verification, recovery codes |
| Dev reload | Air 1.67.4 | `make dev` |
| Lint/security | golangci-lint 2.11.0, govulncheck | `make ci` and `make vulncheck` |
| Deployment | Docker | Multi-stage, non-root Alpine runtime |

### Architecture choices

Choose one stack ID at generation time:

| Stack ID | Use it for |
| --- | --- |
| `go/gin-flat` | Small services and prototypes; one application package |
| `go/gin-layered` | General-purpose API; transport → app → service seams |
| `go/gin-hexagonal` | Boundary-heavy services; domain, ports, adapters |
| `go/gin-ddd` | Complex domains; bounded context with domain/application layers |

All four share the same scaffold contract: `cmd/api`, `internal/httpapi`,
configuration anchors, feature-owned packages, and one overlay family.

### Defaults

- Presets: `api` (default; Postgres + JWT) and `api-custom` (select database,
  cache, queue, scheduler, dedupe, compression, auth, TOTP, media, search,
  search sync, pagination, Docker, and CI).
- Base API: `/healthz`, `/readyz`, `/api/v1/ping`, Swagger UI, merged OpenAPI,
  RFC 9457 errors, request IDs, recovery, body limits, CORS, rate limiting, and
  graceful shutdown.
- Database: explicit goose migrations through `cmd/migrate`; the API never
  migrates on startup.
- Auth: email/password, HS256 access tokens, rotating hashed refresh sessions,
  Google OAuth via Gothic, email verification/reset, and optional TOTP.
- Database workflow: goose migrations embedded in a migrator binary, `sqlc.yaml`
  at the project root, and an opt-in deterministic seed command.
- Skills: all 17 Go skills plus `sql-optimization-patterns` and
  `postgresql-optimization` are installed under `.agents/skills` and
  `.claude/skills`, with `skills-lock.json`, `AGENTS.md`, `CLAUDE.md`, and
  `CONTEXT.md`.

### Project-scoped Go skills

Every generated Go app receives:

- `golang-code-style`, `golang-testing`, `golang-error-handling`,
  `golang-security`, `golang-safety`, `golang-context`, `golang-naming`,
  `golang-documentation`, `golang-design-patterns`,
  `golang-structs-interfaces`, `golang-performance`, `golang-concurrency`,
  `golang-lint`, `golang-benchmark`, `golang-troubleshooting`,
  `golang-spf13-viper`, and `golang-swagger`.

### Overlays

| Overlay | Status | Notes |
| --- | --- | --- |
| `database:postgres` | Implemented | pgx pool, embedded goose migrator, readiness |
| `auth:jwt` | Implemented | Users, HS256, rotating refresh, Google OAuth, SMTP/local email |
| `totp` | Implemented | Enrollment, recovery codes, second-factor login |
| `docker` | Implemented | Multi-stage, non-root, health check |
| `ci:github` | Implemented | Format, vet, lint, tests, race, build, govulncheck |
| `cache:redis` | Implemented | Fail-open cache, TTLs, miniredis tests |
| `queue:rabbitmq` | Implemented | Worker pool, quorum queues, retries, DLQ, scheduler |
| `scheduler:leader` | Implemented | Redis lease with renewal and failover |
| `dedupe:redis` | Implemented | Success markers with fail-closed checks |
| `compression:gzip` | Implemented | Bounded gzip queue payloads |
| `media` | Implemented | Authenticated uploads, private reads, PostgreSQL metadata |
| `media-provider:minio` | Implemented | MinIO, imgproxy, nginx cache |
| `media-provider:r2` | Implemented | Standalone Cloudflare R2, original images only |
| `search:meilisearch` | Implemented | Public query API, internal indexing, Meilisearch container |
| `search-sync:rabbitmq` | Implemented | Queue-backed async indexing with worker retries and DLQ |
| `pagination` | Implemented | Signed HMAC cursors, offset helpers, caps, metadata |
| Observability | Deferred | OTel, Prometheus, Sentry, pprof overlay next |

## Backend options — provisional

| Stack ID (provisional) | Language | Framework | Status |
| --- | --- | --- | --- |
| `rust/axum` | Rust | Axum, SQLx/Postgres, JWT, Docker, CI | Implemented |
| `python/fastapi` | Python | FastAPI candidate | Provisional |

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
| Containers | Docker, multi-stage, non-root | Implemented for Next and Go |
| CI | GitHub Actions | Implemented for Next and Go |
| Database | Postgres 17 | Implemented for Next and Go |
| Cache | Redis | Deferred for Next |
| Queue | RabbitMQ (backend stacks) | Deferred for Next |
| Storage/search | MinIO/imgproxy/nginx, R2, Meilisearch | Implemented for Go; Elasticsearch deferred |
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
