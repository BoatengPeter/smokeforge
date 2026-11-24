# Composable workspace

**Status:** generation implemented; lifecycle changes remain deferred.

`--kind workspace` generates a monorepo from one optional API and any
non-conflicting selection of web, desktop, Android, and iOS clients. Existing
`fullstack`, `mobile`, and `desktop` kinds keep their current behavior.

## Selection

```bash
smokeforge new acme --kind workspace \
  --backend go/gin-layered \
  --frontend next \
  --desktop electron \
  --mobile android/compose \
  --mobile ios/swiftui
```

- `--backend`: optional API stack.
- `--frontend`: optional web stack.
- `--desktop`: optional `electron`, `egui`, or `gpui`.
- `--mobile`: repeatable Android/iOS selection, maximum two.
- At least one lane must be selected.
- `--stack` is rejected; use `--backend` for the API lane.
- Each non-mobile lane is limited to one target. Duplicate mobile stacks are
  rejected.

The `workspace-api` preset supplies the API-backed baseline:
`database:postgres`, `auth:jwt`, and `ci:github`. A client-only workspace starts
without a default preset.

## Targets

Target paths are stable:

| Lane | Path |
| --- | --- |
| API | `apps/api` |
| Web | `apps/web` |
| Desktop | `apps/desktop` |
| Android | `apps/android` |
| iOS | `apps/ios` |

The root target is `workspace/monorepo`. Existing `workspace/fullstack` and
`workspace/mobile` roots remain for their existing kinds.

## Root ownership

The root owns:

- `AGENTS.md` and `CLAUDE.md` routers;
- `CONTEXT.md` and `CONTEXT-MAP.md`;
- `README.md`, `Makefile`, and `.gitignore`;
- `.agents/skills/`, `.claude/skills/`, and `skills-lock.json`;
- cross-client `docs/`.

Root skills are limited to workspace orchestration, API contracts, and
cross-client testing. Platform skills and platform docs stay under each app.

Lane overlays add only selected-lane Make targets, docs sections, context-map
rows, agent routes, and CI jobs. The registry verifier checks that root guidance
does not describe unselected apps as active targets.

## API-less workspaces

When `--backend` is omitted:

- no API target is generated;
- root Make and CI omit API install/dev/test/build;
- API credentials and server secrets are absent;
- clients are standalone targets and do not receive API BFF/auth wiring.

## Feature behavior

- API features route only when an API exists.
- Web, mobile, and desktop features route only to selected lanes.
- `ci` routes to the workspace root.
- Mobile with an API requires API `auth:jwt`.
- API `auth:jwt` adds web BFF and Electron auth wiring when those clients exist.
- API-only features without an API are rejected.

## Verification

`tests/verify-workspace.sh` in the registry generates the all-client shape,
API + desktop without web, client-only, and API-only workspaces. It checks
selected targets, root agent files, skills, lockfiles, docs, Make targets, CI
jobs, and unselected-app absence.

The lifecycle boundary is explicit: adding or removing client targets from an
existing workspace remains a separate project.
