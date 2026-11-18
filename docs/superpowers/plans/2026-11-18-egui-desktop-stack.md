# Egui Desktop Stack Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a native Rust `desktop/egui` stack that mirrors the Electron desktop product surface: a routed shell, local SQLite Library, settings and diagnostics, API authentication, offline outbox sync, manual conflicts, and CI/packaging.

**Architecture:** `eframe` owns the native window and UI loop. A `tokio` runtime owns HTTP, database work, and sync; the UI communicates with background work through typed command/event channels. `rusqlite` + `refinery` own local data. A platform adapter owns keyring, paths, settings, logging, packaging, and optional tray/update behavior. There is no renderer/process boundary like Electron; the Rust code is trusted and calls the API directly.

**Tech Stack:** Rust 1.96, eframe/egui 0.37, egui_extras 0.37, egui_tiles 0.18, egui_plot 0.38, egui_notify 0.24, egui_form 0.11, egui-file-dialog 0.16, Tokio 1.54, reqwest 0.13, rusqlite 0.43, r2d2_sqlite 0.37, refinery 0.10, keyring 4.2, tracing 0.1, directories 6, garde 0.23, rust-i18n 4.2, cargo-packager 0.11.

---

## Locked decisions

- Use egui 0.37, not egui 0.36. Reject `egui_dock 0.21` and `egui-phosphor 0.14` because they pin egui 0.36. Use `egui_tiles` and embedded Lucide SVGs instead.
- Keep native window decorations in v1. Custom window chrome is a later optional feature; egui/winit custom controls add platform-specific accessibility risk.
- Use `rusqlite` with `r2d2_sqlite` and `refinery` rather than SQLx. The database is local, synchronous, and simpler to package without a database URL or SQLx offline metadata.
- Use `keyring` for refresh tokens. Never store tokens in SQLite or settings JSON.
- Use manual conflict resolution by default, matching the Electron desktop.
- Fullstack egui uses the same server sync contract as Electron: versions, `If-Match`, `Idempotency-Key`, changefeed cursors, and tombstones.
- The standalone egui preset is local-only. Fullstack desktop enables sync.
- In-app command palette and keyboard registry are in v1. Native menu/tray are opt-in platform features.
- Accessibility is best-effort: egui does not expose a full browser-grade accessibility tree. Document the limitation and implement keyboard navigation, focus rings, high-contrast mode, and scalable UI.

## File structure

```text
templates/desktop/egui/
  template.yaml
  files/
    Cargo.toml.tmpl
    rust-toolchain.toml
    Makefile.tmpl
    README.md.tmpl
    CONTEXT.md
    AGENTS.md
    DESIGN.md
    Packager.toml.tmpl
    src/
      main.rs
      app.rs
      app_state.rs
      events.rs
      commands.rs
      i18n.rs
      theme.rs
      paths.rs
      config.rs
      logging.rs
      settings.rs
      db/
        mod.rs
        migrations.rs
        library_repository.rs
        sync_outbox.rs
      api/
        mod.rs
        client.rs
        auth.rs
        library.rs
      sync/
        mod.rs
        engine.rs
        conflict.rs
      platform/
        mod.rs
        keyring.rs
        window.rs
        tray.rs
        menu.rs
        notifications.rs
        updater.rs
      ui/
        shell.rs
        sidebar.rs
        top_bar.rs
        status_bar.rs
        command_palette.rs
        components.rs
        icons.rs
        dashboard.rs
        library.rs
        activity.rs
        settings.rs
        account.rs
        about.rs
        onboarding.rs
    assets/
      icons/*.svg
      migrations/0001_init.sql
      migrations/0002_sync.sql
    tests/
      app_smoke.rs
      library_db.rs
      sync_engine.rs
      conflict.rs
      fixtures/
overlays/desktop/egui/
  testing/e2e/
  shell/tray/
  shell/native-menu/
  updates/self-update/
```

Feature overlays are intentionally few. The base egui stack includes the shell, local database, API client, sync engine, settings, and conflict UI. Optional overlays add platform integrations and tests.

## Clauses and compatibility checks

Run these before creating files:

```bash
cargo search eframe --limit 1
cargo info eframe@0.37.0
cargo info egui_tiles@0.18.0
cargo info egui_plot@0.38.0
cargo info egui_form@0.11.0
cargo info egui-file-dialog@0.16.0
cargo info rusqlite@0.43.0
cargo info refinery@0.10.0
cargo info cargo-packager@0.11.8
```

Expected: all resolve against `egui 0.37`. If a dependency moves to a new incompatible egui version, pin the last compatible version in `Cargo.toml.tmpl`; do not mix egui majors.

## Task 1: Scaffold the stack and CLI integration

**Files:**
- Create: `templates/desktop/egui/template.yaml`
- Create: `templates/desktop/egui/files/Cargo.toml.tmpl`
- Create: `templates/desktop/egui/files/src/main.rs`
- Create: `templates/desktop/egui/files/src/app.rs`
- Modify: `smokeforge-registry/manifest.yaml`
- Modify: `smokeforge/internal/resolve/resolve.go`
- Modify: `smokeforge/internal/resolve/resolve_test.go`
- Modify: `smokeforge/internal/cli/new.go` only if stack selection needs help text

**Steps:**
- [ ] Add stack `desktop/egui` with variables `crate_name`, `bundle_id`, `api_url`.
- [ ] Add `desktop-egui` preset that selects the stack and no JavaScript-only features.
- [ ] Add resolver validation so fullstack desktop accepts `electron` and `egui`.
- [ ] For fullstack egui, add `sync:library` to the API feature list but do not add `auth:electron`, `state:zustand`, or `storage:sqlite` desktop overlays.
- [ ] Write the minimal eframe window:

```rust
fn main() -> eframe::Result<()> {
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default()
            .with_inner_size([1180.0, 760.0])
            .with_min_inner_size([760.0, 520.0])
            .with_title("{{.project_name}}"),
        ..Default::default()
    };

    eframe::run_native(
        "{{.project_name}}",
        options,
        Box::new(|cc| Ok(Box::new(app::DesktopApp::new(cc)))),
    )
}
```

- [ ] Add a smoke test that constructs `DesktopApp` with `egui_kittest` and asserts the app name renders.
- [ ] Run `cargo fmt --check`, `cargo clippy --all-targets -- -D warnings`, and `cargo test`.
- [ ] Commit: `feat(egui): scaffold native desktop stack`.

## Task 2: App state, events, commands, and theme

**Files:**
- Create: `src/app_state.rs`
- Create: `src/events.rs`
- Create: `src/commands.rs`
- Create: `src/theme.rs`
- Create: `src/ui/shell.rs`
- Create: `src/ui/sidebar.rs`
- Create: `src/ui/top_bar.rs`
- Create: `src/ui/status_bar.rs`
- Create: `src/ui/command_palette.rs`
- Create: `src/ui/components.rs`
- Create: `src/ui/icons.rs`

**Steps:**
- [ ] Define `AppView` as `Dashboard`, `Library`, `Activity`, `Settings`, `Account`, `About`; restore the last view from settings.
- [ ] Define `AppEvent` for `AuthChanged`, `LibraryChanged`, `SyncStatus`, `JobProgress`, `Toast`, and `ShutdownRequested`.
- [ ] Define the command registry with stable IDs and accelerators. Reuse the Electron command vocabulary and native-menu IDs.
- [ ] Load Lucide SVG assets through `egui_extras` image loaders. Do not introduce an icon crate that pins egui 0.36.
- [ ] Implement the shared brand palette as Rust theme tokens using the same cream, charcoal, ink, tide, coral, sand, mist, and bark values as `desktop/electron/DESIGN.md`.
- [ ] Implement a theme selector for light, dark, and system. Use `egui::Visuals` and persist the preference.
- [ ] Implement `SidePanel`, `TopBottomPanel` status bar, and `CentralPanel` content.
- [ ] Implement command palette with a modal `Window`, `TextEdit`, substring filtering, arrow navigation, and Enter execution.
- [ ] Implement keyboard shortcuts for Cmd/Ctrl+K, Cmd/Ctrl+B, Cmd/Ctrl+1/2/3, and Cmd/Ctrl+,.
- [ ] Add `egui_kittest` tests for navigation, sidebar collapse, command filtering, and theme switching.
- [ ] Commit: `feat(egui): add app shell, commands, and theme`.

## Task 3: Settings, paths, logging, keyring, and window state

**Files:**
- Create: `src/paths.rs`
- Create: `src/config.rs`
- Create: `src/settings.rs`
- Create: `src/logging.rs`
- Create: `src/platform/keyring.rs`
- Modify: `src/app.rs`
- Create: `tests/settings_test.rs`

**Steps:**
- [ ] Use `directories::ProjectDirs` for config, data, cache, and log paths.
- [ ] Implement an atomic JSON settings store with `serde`, `tempfile`, and rename; quarantine corrupt files.
- [ ] Store only non-sensitive preferences: theme, language, sidebar width, last view, window bounds, sync policy, update channel.
- [ ] Store refresh tokens only through `keyring`. If the keyring is unavailable, keep them in memory and document that they will not survive restart.
- [ ] Implement `tracing` with a file appender and a redaction layer for `token`, `access_token`, `refresh_token`, `password`, and `secret`.
- [ ] Persist and restore window size and position with bounds validation.
- [ ] Add tests for defaults, atomic writes, corrupt JSON recovery, and invalid window bounds.
- [ ] Commit: `feat(egui): add settings, logging, keyring, and window state`.

## Task 4: Local SQLite, migrations, Library CRUD, and virtualized table

**Files:**
- Create: `src/db/mod.rs`
- Create: `src/db/migrations.rs`
- Create: `src/db/library_repository.rs`
- Create: `src/db/sync_outbox.rs`
- Create: `assets/migrations/0001_init.sql`
- Create: `assets/migrations/0002_sync.sql`
- Create: `src/ui/library.rs`
- Create: `tests/library_db.rs`

**Steps:**
- [ ] Add `0001_init.sql` with `local_kv`, `library_items`, indexes, and `user_version`/refinery metadata.
- [ ] Add `0002_sync.sql` with `version`, `sync_status`, `owner_id`, `sync_outbox`, `sync_cursor`, and `sync_conflicts`.
- [ ] Use `r2d2_sqlite` and `refinery` embedded migrations. Run migrations once at startup on a background thread.
- [ ] Implement repository methods `list`, `get`, `create`, `update`, `soft_delete`, `restore`, `import`, `export`, and `backup`.
- [ ] Use stable client-generated UUIDs and `version = 0` until the first server acknowledgement.
- [ ] Implement the list/detail layout with a resizable `SidePanel::right` inspector.
- [ ] Use `egui_extras::TableBuilder` with striped rows, search, status filter, sorting, and virtualized row body.
- [ ] Implement create/edit forms with `egui_form` and `garde` validation.
- [ ] Implement delete with an undo action in `egui_notify`.
- [ ] Add a conflict row state and manual conflict panel with Keep local and Use server.
- [ ] Add database tests for migrations, CRUD, search, soft delete, restore, and import duplicate titles.
- [ ] Commit: `feat(egui): add local Library and virtualized table`.

## Task 5: API client and fullstack authentication

**Files:**
- Create: `src/api/mod.rs`
- Create: `src/api/client.rs`
- Create: `src/api/auth.rs`
- Create: `src/api/library.rs`
- Modify: `src/app.rs`
- Modify: `src/config.rs`
- Create: `tests/api_client.rs`

**Steps:**
- [ ] Create a `reqwest::Client` with JSON, rustls, timeout, and system proxy support.
- [ ] Add `ApiError` mapping from RFC 9457 problem responses to stable codes/messages.
- [ ] Implement `AuthClient` for `/api/v1/auth/login`, `register`, `refresh`, `logout`, and `me`.
- [ ] Keep the access token in memory and the refresh token in the OS keyring.
- [ ] Implement one refresh-and-retry on `401`.
- [ ] Implement `LibraryApi` for snapshot, changes, create, update, and delete with the sync headers.
- [ ] Run auth and API calls on the Tokio runtime; send results as `AppEvent` to the UI.
- [ ] Use `wiremock` tests for login, refresh, RFC 9457 errors, idempotency headers, and version headers.
- [ ] Commit: `feat(egui): add API client and keyring auth`.

## Task 6: Offline sync engine and manual conflicts

**Files:**
- Create: `src/sync/mod.rs`
- Create: `src/sync/engine.rs`
- Create: `src/sync/conflict.rs`
- Modify: `src/db/sync_outbox.rs`
- Modify: `src/ui/library.rs`
- Create: `tests/sync_engine.rs`
- Create: `tests/conflict.rs`

**Steps:**
- [ ] On every local mutation, write the item and outbox row in one SQLite transaction.
- [ ] Implement ordered single-flight outbox processing.
- [ ] Send `Idempotency-Key` and `If-Match` for all mutations.
- [ ] Pull changefeed pages by cursor and apply tombstones; skip entities with pending outbox rows.
- [ ] Retry network, `408`, `429`, and `5xx` with exponential backoff, jitter, a five-minute cap, and 10 attempts; honor `Retry-After`.
- [ ] On `409`, fetch the server snapshot, record a conflict, and stop automatic retry for that row.
- [ ] Implement manual resolution: Keep local re-queues against the server version with a new key; Use server replaces the local row and deletes the outbox row.
- [ ] Show pending, synced, failed, and conflict status in the Library and status bar.
- [ ] Test offline mutation, reconnect, retry, duplicate idempotency key, stale version, and both manual resolutions.
- [ ] Commit: `feat(egui): add offline outbox and manual conflicts`.

## Task 7: Activity, dashboard, Settings, Account, and About

**Files:**
- Create: `src/ui/dashboard.rs`
- Create: `src/ui/activity.rs`
- Create: `src/ui/settings.rs`
- Create: `src/ui/account.rs`
- Create: `src/ui/about.rs`
- Create: `src/ui/onboarding.rs`
- Modify: `src/app_state.rs`
- Modify: `src/app.rs`

**Steps:**
- [ ] Dashboard: stat cards, `egui_plot` chart, recent Library items, empty/loading states.
- [ ] Activity: a background worker over a bounded Tokio channel with progress, cancel, and retry; display in the page and status bar.
- [ ] Settings: theme, language, launch-at-login where the platform supports it, sync policy, database path, backup/export/import, log level, update channel.
- [ ] Account: API login, register, signed-in state, logout, and sync start/stop.
- [ ] About: versions, platform, paths, copy diagnostics, export diagnostics, open log folder.
- [ ] Onboarding: first-run dialog; what's-new after version change; persisted completion flags.
- [ ] Add `egui_kittest` tests for onboarding, settings persistence, account signed-out state, and diagnostics copy.
- [ ] Commit: `feat(egui): add product pages and onboarding`.

## Task 8: Native integration and packaging

**Files:**
- Create: `src/platform/menu.rs`
- Create: `src/platform/tray.rs`
- Create: `src/platform/notifications.rs`
- Create: `src/platform/updater.rs`
- Create: `Packager.toml.tmpl`
- Modify: `Makefile.tmpl`
- Create: `docs/PACKAGING.md`
- Create: `docs/UPDATES.md`
- Create: `docs/TRAY.md`

**Steps:**
- [ ] Spike `muda` and `tray-icon` against eframe 0.37's winit loop; if the event bridge is unreliable, ship in-app menus in v1 and keep native menu/tray as opt-in overlays.
- [ ] Add native notifications with permission-aware failure handling.
- [ ] Add a tray feature with show/hide, New Library item, Settings, and Quit.
- [ ] Add optional `self_update` support with a disabled default until signing and a publish channel exist.
- [ ] Add `cargo-packager` configuration for macOS DMG, Windows MSI/NSIS, Linux AppImage/deb.
- [ ] Document code signing, notarization, update metadata, and platform limits.
- [ ] Add packaging smoke checks on Linux.
- [ ] Commit: `feat(egui): add native integration and packaging`.

## Task 9: Documentation, CI, and release

**Files:**
- Create: `docs/LIBRARIES.md`
- Create: `docs/UI.md`
- Create: `docs/CONTEXT-MAP.md` if needed
- Modify: `CONTEXT.md`, `AGENTS.md`, `README.md.tmpl`, `DESIGN.md`
- Modify: `smokeforge-registry/manifest.yaml`
- Modify: `smokeforge/README.md`, `STACKS.md`, `CHANGELOG.md`
- Modify: `.github/workflows/ci.yml`

**Steps:**
- [ ] Document crate choices, egui limitations, native integration risks, and the sync contract.
- [ ] Add `egui` to stack tables, feature tables, examples, and release notes.
- [ ] Add CI jobs:
  - `cargo fmt --check`
  - `cargo clippy --all-targets -- -D warnings`
  - `cargo test`
  - `cargo check` on Ubuntu, Windows, macOS
  - offline/sync integration with Postgres and a live API
  - `egui_kittest` snapshot tests
  - Linux packaging smoke
- [ ] Add a generated `desktop/egui` smoke app to `verify-desktop` or a dedicated `verify-egui`.
- [ ] Run the live desktop-to-API sync E2E against Go and Rust.
- [ ] Document that the UI must be verified at 760×520 and 1440×900.
- [ ] Commit: `docs(egui): document native stack and CI`.

## Verification gates

Every task ends with:

```bash
cargo fmt --all -- --check
cargo clippy --all-targets -- -D warnings
cargo test
```

For generated-app checks:

```bash
smokeforge new /tmp/egui-smoke --registry . --kind desktop --stack desktop/egui --preset desktop-egui --no-input --no-git
cd /tmp/egui-smoke
cargo check
cargo test
```

Fullstack sync gate:

```bash
# Start temporary Postgres on an isolated port, migrate/seed the Go or Rust API,
# then run the generated desktop sync test with:
SYNC_E2E_API_URL=http://localhost:18080 \
SYNC_E2E_EMAIL=demo@example.com \
SYNC_E2E_PASSWORD=... \
cargo test --test live_sync
```

Expected: local Library mutation appears in the API snapshot; a stale version produces a manual conflict; client-wins/server-wins choices resolve it.

## Risks and non-goals

- egui is not a browser; accessibility is reduced compared with Radix/React. Do not claim browser-grade screen-reader parity.
- `egui_dock`, `egui-phosphor`, and `egui-toast` currently target egui 0.36. Do not force them into 0.37; use compatible alternatives.
- Native menus/tray may require a separate winit event bridge. Prototype before committing to the default shell.
- Custom title bars are deferred; native decorations are the safe default.
- Webview embedding is out of scope. Use native egui widgets.
- Marketing site and mobile clients are out of scope.
- The existing Electron stack remains supported; egui is a second desktop stack, not a replacement.

## Execution order

1. Scaffold and CLI/manifest integration.
2. Shell, theme, commands, settings, logging, keyring.
3. SQLite Library and virtualized table.
4. API auth and sync engine + manual conflicts.
5. Product pages and onboarding.
6. Native integration, packaging, updates.
7. Documentation, CI matrix, live API E2E, release.
