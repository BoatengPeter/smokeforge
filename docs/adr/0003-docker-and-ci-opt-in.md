# 0003 — Docker and CI are opt-in overlays

- Status: Accepted
- Date: 2026-11-12

## Context

Generated apps target different environments: Vercel, static hosting, Node
servers, or Docker. CI providers also vary. Forcing container and workflow files
into every project adds unused files and assumptions.

## Decision

Docker and CI are opt-in features, not base defaults.

- `docker` adds a package-manager-aware multi-stage Dockerfile, `.dockerignore`,
  compose configuration, and Make targets.
- `ci:github` adds a GitHub Actions workflow that runs install, lint, typecheck,
  tests, build, and e2e.
- Users opt in through the wizard or `--feature docker --feature ci:github`.
- The base app includes only application files and `.gitignore`.
- A future bundle/preset may select both together, but individual selection
  remains possible.

## Consequences

- Projects that do not need containers stay lean.
- Database/local-service compose behavior must coordinate with the database
  overlay.
- CI providers other than GitHub require a separate variant or overlay.
- Documentation must show opt-in flags and generated commands.
