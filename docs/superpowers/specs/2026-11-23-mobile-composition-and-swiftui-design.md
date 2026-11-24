# Mobile Composition and SwiftUI Design

**Status:** Approved design
**Date:** 2026-11-23
**Repositories:** `smokeforge` (CLI), `smokeforge-registry`

## Goal

Add a production-shaped SwiftUI iOS template and make smokeforge mobile
composition explicit: standalone mobile apps, API + mobile workspaces, and
fullstack projects with web and one or both mobile clients.

## Project shapes

| Command | Result |
| --- | --- |
| `--kind mobile --stack ios/swiftui` | iOS app only |
| `--kind mobile --stack android/compose` | Android app only |
| `--kind mobile --stack ios/swiftui --backend go/gin-layered` | API + iOS, no web |
| `--kind fullstack --backend X --mobile ios/swiftui` | API + iOS, no web |
| `--kind fullstack --backend X --mobile android/compose --mobile ios/swiftui` | API + both mobile apps, no web |
| `--kind fullstack --backend X --frontend next` | API + web |
| `--kind fullstack --backend X --frontend next --mobile ...` | API + web + selected mobile apps |

### Composition rules

- `--mobile` is repeatable, up to the manifest's mobile limit of two.
- Fullstack requires a backend and at least one client target.
- Web is generated only when `--frontend next` is explicit. This is a breaking
  behavior change from the previous web-default fullstack behavior.
- `--kind mobile --backend` generates a multi-target API + mobile project using
  `workspace/mobile`.
- Fullstack without web uses `workspace/mobile`.
- Fullstack with web uses `workspace/fullstack`; Android/iOS selections add the
  corresponding mobile workspace overlays.
- Backend feature overlays apply to the API target in both shapes.
- Mobile targets require API `auth:jwt` in API + mobile and fullstack-mobile
  compositions.
- Add a `mobile-api` preset for API + mobile (Postgres, JWT, Docker, CI).
  `fullstack-api` is valid only when web is selected.

## Workspace roots

- Keep `workspace/fullstack` for web + API, optionally with mobile overlays.
- Add `workspace/mobile` for API + one or both mobile targets.
- `workspace/mobile` owns its README, Makefile, context map, environment docs,
  and CI. It does not reference web unless web is selected.
- Generated target paths remain `apps/api`, `apps/web`, `apps/android`, and
  `apps/ios`.
- Existing fullstack mobile overlays become target-aware so Android and iOS can
  coexist without overwriting each other.

## iOS template

### Toolchain

| Component | Choice |
| --- | --- |
| Project generation | XcodeGen 2.48.0; `project.yml` is source of truth |
| Xcode | 26.6 stable on macOS 26 CI |
| SDK | iOS 26 |
| Minimum iOS | 17 |
| Swift | 6 language mode, strict concurrency |
| UI | SwiftUI only; UIKit only through explicit interop |
| Networking | URLSession async/await |
| Tokens | Keychain |
| Tests | Swift Testing |

No `.xcodeproj` is committed. Generated apps run `make project` to generate it.
XcodeGen cannot run on Linux; all iOS verification happens on macOS CI.

### Modules

```text
App/
Packages/
  Core/
  Networking/
  Storage/
  Data/
  DesignSystem/
  TestSupport/
  AuthFeature/
  HomeFeature/
  SettingsFeature/
```

- `Core` owns pure models, result/error types, and shared contracts.
- `Networking` owns URLSession API clients, DTOs, auth request headers, refresh
  rotation, and RFC 9457 mapping.
- `Storage` owns Keychain access and non-sensitive preferences.
- `Data` owns repositories and session policy.
- `DesignSystem` owns the Smoke Hex theme and shared SwiftUI components.
- `TestSupport` owns fakes and fixtures.
- Feature packages own views, ViewModels, and feature routes. Features never
  depend on each other.
- The thin app target owns `@main`, the root coordinator, the session gate, and
  dependency wiring.

### Architecture

- MVVM with `@Observable`, `@MainActor` view models, and initializer injection.
- Typed `NavigationStack` path in the app coordinator.
- Unidirectional data flow: immutable view state, events upward, state downward.
- View models do not import UIKit or SwiftUI.
- Use value types by default; share only through observable main-actor models.
- Main actor by default for app/UI packages; networking and storage remain
  nonisolated and testable.
- Prefer simple concrete types before protocols; introduce protocols only for
  test substitution.

### Features

- Email/password registration, login, native refresh rotation, logout, and
  current-account loading.
- Access and refresh tokens stored in Keychain with a device-only accessibility
  class.
- Auth shell: sign in, create account, forgot-password request.
- Home: API reachability and account summary.
- Settings: environment, API base URL, appearance, and sign out.
- No SwiftData, offline outbox, OAuth, TOTP, analytics, or crash reporting.
- OAuth, TOTP, and offline sync remain overlays.

### Platform settings

- Debug, Staging, and Release xcconfigs and bundle ID suffixes.
- Debug API default `http://127.0.0.1:8080`.
- Native SwiftUI controls; no custom chrome that fights system design.
- Info.plist, privacy manifest, launch screen, String Catalog, accessibility
  labels, Dynamic Type, and an adaptive phone/tablet layout.
- Smoke Hex app icon and palette.

### Agent files and skills

Generated apps receive `AGENTS.md`, `CLAUDE.md`, `CONTEXT.md`, `DESIGN.md`,
`docs/`, and project-scoped skills:

- `write-swift`
- `swiftui-pro`
- `swiftui-expert-skill`

The skills are copied under `.agents/skills` and `.claude/skills` and recorded
in `skills-lock.json`.

## CI

- Add a registry `verify-ios` job on `macos-26` with Xcode 26.6.
- Install XcodeGen 2.48.0 and generate the project.
- Run Swift package tests, app unit tests, and a simulator build/test using an
  installed iPhone 17 simulator.
- Add an iOS job to generated `workspace/mobile` and fullstack CI templates.
- No local Xcode verification is possible in this environment.

## Delivery order

1. CLI multi-mobile and optional-web composition.
2. `workspace/mobile` and target-aware mobile overlays.
3. iOS SwiftUI template and XcodeGen configuration.
4. iOS agent files, docs, skills, and macOS CI.
5. Generated layout verification and CLI tests.
6. Push, then verify iOS on GitHub macOS CI.

## Out of scope

- SwiftData, offline sync, and background mutation queues.
- OpenAPI-generated Swift clients.
- OAuth, TOTP, Sign in with Apple, passkeys, push, analytics, crash reporting,
  App Store signing, TestFlight, and phased release automation.
- Snapshot tests, XCUITest suites, and performance benchmarks.
