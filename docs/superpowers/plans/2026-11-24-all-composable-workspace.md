# All-Composable Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--kind workspace` with optional API, web, desktop, Android, and iOS lanes, plus a root agent/skill/documentation surface.

**Architecture:** Keep existing kinds untouched. Add a workspace resolver with optional API, a `workspace/monorepo` root base, and one synthetic root overlay per selected lane. Root owns cross-cutting skills/docs; app skills stay app-local.

**Tech Stack:** Go CLI, registry YAML/templates, shell verification, Go tests, GitHub Actions.

---

### Task 1: Workspace resolver and plan model

**Files:**
- Modify: `internal/resolve/resolve.go`
- Modify: `internal/resolve/resolve_test.go`
- Modify: `internal/cli/new.go`
- Test: `internal/resolve/resolve_test.go`

- [ ] Add failing tests for all-client, desktop-without-web, API-less, duplicate-lane, and no-target workspace selections.
- [ ] Run `go test ./internal/resolve` and confirm the tests fail because `workspace` is unknown.
- [ ] Add a `Plan.API *TargetPlan` field and dispatch `kind == "workspace"` to `resolveWorkspace`.
- [ ] Implement `workspaceStacks`, `workspaceVariables`, `routeCompositionFeatures`, `validateWorkspaceFeatures`, and `buildWorkspacePlan`.
- [ ] Append root lane overlays from `overlays/workspace/monorepo/{api,web,mobile,desktop-*}`.
- [ ] Run `go test ./internal/resolve` until green.

### Task 2: CLI workspace flags, presets, summaries

**Files:**
- Modify: `internal/cli/new.go`
- Modify: `internal/cli/new_test.go`
- Test: `internal/cli/new_test.go`

- [ ] Add failing tests for workspace validation, optional API, desktop-without-web, and `workspace-api` default preset.
- [ ] Extend help/examples and flags for `workspace`.
- [ ] Allow `--desktop`, `--mobile`, `--frontend` with workspace; reject `--stack`.
- [ ] Add shape-aware `workspace-api` default when API exists.
- [ ] Make summary and next steps tolerate `Plan.API == nil`.
- [ ] Run `go test ./internal/cli`.

### Task 3: Architecture output

**Files:**
- Modify: `internal/engine/architecture.go`
- Modify: `internal/engine/architecture_test.go`

- [ ] Add failing tests for a workspace root with API and clients, and an API-less workspace.
- [ ] Add workspace diagram and root directory-map rows for agents, skills, docs, and workspace files.
- [ ] Run `go test ./internal/engine`.

### Task 4: Registry root base

**Files:**
- Create: `templates/workspace/monorepo/template.yaml`
- Create: `templates/workspace/monorepo/files/...`
- Modify: `manifest.yaml`

- [ ] Add `workspace` kind, `workspace/monorepo` stack, and `workspace-api` preset.
- [ ] Add root `README`, `Makefile`, `CONTEXT`, `CONTEXT-MAP`, `AGENTS`, `CLAUDE`, `.gitignore`.
- [ ] Add root cross-cutting skills, `skills-lock.json`, and `docs/`.
- [ ] Add root CI and lane overlays.
- [ ] Add manifest root feature wiring for `ci` and workspace lanes.
- [ ] Run registry YAML validation.

### Task 5: Lane overlays and CI

**Files:**
- Create: `overlays/workspace/monorepo/api/overlay.yaml`
- Create: `overlays/workspace/monorepo/web/overlay.yaml`
- Create: `overlays/workspace/monorepo/mobile/overlay.yaml`
- Create: `overlays/workspace/monorepo/desktop-*/overlay.yaml`
- Create: `overlays/workspace/monorepo/ci/`

- [ ] Add selected-lane Make targets, install/test/ci/clean hooks, README sections, context-map rows, and agent routing.
- [ ] Add conditional CI jobs for API/web/Electron/egui/GPUI/Android/iOS.
- [ ] Verify `make -n` for representative generated workspaces.

### Task 6: Verification matrix

**Files:**
- Create: `tests/verify-workspace.sh`
- Modify: `tests/README.md`
- Modify: `.github/workflows/ci.yml`

- [ ] Generate all-client, desktop-without-web, iOS-only, Android-only, web-only, and client-only workspaces.
- [ ] Assert selected targets, root agents/skills/docs, Make targets, CI jobs, and no references to unselected apps.
- [ ] Run the script locally where possible and add CI coverage.

### Task 7: Documentation and commit

**Files:**
- Modify: `README.md`, `STACKS.md`, `docs/architecture/fullstack.md`, `docs/architecture/cli.md`, `docs/architecture/workspace.md`, `docs/development/roadmap.md`

- [ ] Document `--kind workspace`, optional lanes, root skills/docs, and generation/lifecycle boundaries.
- [ ] Run `go test ./...`, `golangci-lint run ./...`, registry YAML/plist/shell checks.
- [ ] Commit CLI and registry changes separately; do not push.
