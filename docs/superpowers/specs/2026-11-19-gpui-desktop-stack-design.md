# GPUI Desktop Stack Design

**Status:** Approved  
**Date:** 2026-11-19  
**Stack:** `desktop/gpui`  
**Preset:** `desktop-gpui`

## Goal

Add a native GPUI desktop stack that mirrors the egui product boundary while using the GPUI Kit component system. It must support a local-only standalone mode and a reserved fullstack `--desktop gpui` mode.

## Dependencies

Use `gpui-kit = "=0.10.0"` as the only UI dependency. It pins the GPUI snapshot exactly. Do not add raw Zed Git dependencies, `gpui 0.2.2`, direct `gpui-pre-platform`, `gpui-component` revision pins, or `gpui-fast` in v1.

The generated `Cargo.lock` must contain the GPUI Kit-compatible snapshot `gpui-pre 0.3.12`; CI must reject a mixed or newer snapshot.

Install these project-scoped skills in the generated app:

- `gpui-kit` — setup, component catalog, entities, actions, async, focus, testing, coding guides.
- `gpui-kit-design-guides` — layout, density, hierarchy, interaction states, overlays, copy, accessibility.

Vendor the same Rust skills as egui: `rust-async-patterns`, `rust-patterns`, `rust-testing`, `rust-best-practices-apollo`, and `rust-best-practices-beautifulrust`.

Do not create a GPUI-specific copy of the egui component skill. Use the official GPUI Kit skills, `AGENTS.md`, `CONTEXT.md`, and `docs/UI.md` for project rules.

## Workspace

```text
crates/core/        domain and validation; no GPUI
crates/api-client/  reqwest/rustls API client; no GPUI
crates/storage/     rusqlite/refinery, settings, outbox; no GPUI
crates/ui/          GPUI Kit theme tokens, icons, reusable components
crates/app/         entities, models, views, actions, keybindings, async state
crates/desktop/     binary, application bootstrap, assets, logging, panic hook
assets/             icons, fonts, images
docs/               design, UI, auth, sync, packaging, updates, crash
```

Only `ui`, `app`, and `desktop` may depend on `gpui-kit`.

## Bootstrap and Shell

- Use `gpui_kit::application().with_assets(gpui_kit::assets::Assets)`.
- Call `gpui_kit::init(cx)` before any component.
- Open windows with `gpui_kit::open_window(...)`; do not use old `Application::new` patterns.
- Use GPUI Kit `Root`, `Sidebar`, `TitleBar`, `StatusBar`, `Command`, `Dialog`, `Resizable`, `DataTable`, `VirtualList`, and `Notification`.
- Build a fixed shell with Dashboard, Library, Activity, Settings, Account, and About.
- Standalone hides Account; API mode shows it.
- Use semantic `cx.theme()` tokens. No raw colors in feature views.
- Support light/dark themes and a high-contrast token set.
- Use typed `smokeforge` actions and one keybinding registry.

## State and Async

- Models own state and business rules; views implement `Render` and subscribe.
- Store `Subscription` and `Task` handles; never leak them.
- `render` must not perform I/O, locks, network calls, or heavy transformations.
- Use `background_spawn` for CPU or blocking work.
- Use a dedicated Tokio runtime bridge for reqwest and other Tokio-based clients; return typed results to models on the GPUI foreground executor.
- Represent resources as explicit `Loading | Ready | Failed` states.
- Use domain-derived `ElementId`s, never list indexes.
- Public APIs use builders and reader methods, not public fields across crate seams.

## Data and Sync

- Reuse the egui local contract: WAL SQLite, refinery migrations, per-user paths, validated CRUD, soft delete/undo, import/export, backup, clear local data.
- Reuse the sync contract: transactional outbox, `Idempotency-Key`, `If-Match`, changefeed cursor, tombstones, jittered retries, manual conflicts, Keep local / Use server.
- Refresh tokens go to the OS keyring; access tokens stay in memory.
- Standalone seeds demo rows and never starts sync.
- Fullstack enables API mode and sync; the GPUI view layer never calls storage or HTTP directly.

## Fullstack

Reserve `--desktop gpui`. The fullstack resolver adds `apps/desktop` as a GPUI target and applies only a workspace wiring overlay. It must not apply Electron or egui overlays.

## Testing and Quality

- Plain tests for `core`, `storage`, and `api-client`.
- `#[gpui_kit::test]` with `test-support` for entities, actions, command palette, account states, and subscriptions.
- Use deterministic GPUI executors, not sleeps.
- CI: generated-app fmt, clippy, tests, pinned-snapshot check, then Linux build; macOS/Windows compile checks follow.
- Native accessibility requires screen-reader verification before claiming parity.

## Packaging

- `cargo-packager` metadata with AppImage/deb on Linux first.
- Add macOS DMG/signing/notarization and Windows MSI/signing configs on native runners.
- No auto-update in v1.
- Document native build dependencies, assets, icons, panic handling, and platform limits.

## Implementation Slices

1. Scaffold stack, workspace, pins, skills, minimal window.
2. Shell, theme, navigation, command palette, actions, i18n.
3. Settings, logging, keyring, single instance, crash handling.
4. Local Library storage, tables, forms, import/export/backup.
5. API client, Account, auth, token lifecycle.
6. Offline sync, cursor pull, retries, conflicts, conflict UI.
7. Dashboard, Activity, onboarding, full settings/About.
8. Fullstack `--desktop gpui` overlay.
9. Packaging, docs, CI, accessibility and release notes.
