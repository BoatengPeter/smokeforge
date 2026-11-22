# Android Compose Production Stack

**Status:** Approved design
**Date:** 2026-11-20
**Repositories:** `smokeforge` (CLI), `smokeforge-registry`

## Goal

Add a production-grade Android stack to smokeforge. The generated app must be
usable as a standalone mobile project or as the `apps/android` target of a
fullstack workspace, and it must provide a credible foundation for mature
production applications rather than a demo shell.

## Decisions

- Native Android with Kotlin and Jetpack Compose.
- Single-activity app with unidirectional data flow.
- Modular Gradle project with convention plugins.
- Hilt with KSP for dependency injection.
- Retrofit and OkHttp with `kotlinx.serialization`.
- Navigation Compose, not Navigation 3, until its stability and toolchain
  support are verified against the pinned Compose release.
- DataStore plus Android Keystore-backed token encryption.
- Email/password registration, login, refresh, logout, and current account.
- Google OAuth and TOTP are future mobile overlays.
- Offline storage and sync are a future overlay, not part of the base.
- Production foundation includes debug, staging, and release variants, R8,
  staging API configuration, redacted logging, and release signing from
  environment variables.
- Crash reporting, analytics, Play delivery, baseline profiles, screenshot
  tests, and deep architecture tests are future overlays.

## Module structure

```text
app
build-logic
core:common
core:model
core:designsystem
core:network
core:datastore
core:data
core:testing
feature:auth
feature:home
feature:settings
```

### Boundaries

- `:app` owns `Application`, `MainActivity`, the session gate, root navigation,
  and bottom navigation. It is the composition root for feature destinations.
- `:core:common` owns dispatchers, result types, and cross-cutting pure utilities.
- `:core:model` owns shared domain models and session/user types. It contains no
  Android framework imports.
- `:core:designsystem` owns the Smoke Hex theme, colour and type tokens, shared
  components, and previews.
- `:core:network` owns Retrofit services, DTOs, interceptors, token refresh, and
  API error mapping.
- `:core:datastore` owns preferences and the encrypted token store. It does not
  own session policy.
- `:core:data` owns repositories, session state, refresh policy, and Hilt
  bindings between network and storage.
- `:core:testing` owns fakes, test data, and test rules. It is test-only.
- `:feature:*` owns screen state, ViewModels, and Compose UI. Features never
  depend on other features.
- Cross-feature navigation uses callbacks supplied by `:app`.

## Android application

### Screens

- Authentication: sign in, create account, forgot-password request.
- Home: API reachability, signed-in account summary, retry and offline error
  states.
- Settings: environment, API base URL, theme preference, and logout.

### UI rules

- Material 3 with Smoke Hex tokens.
- Edge-to-edge layout and window-size-aware spacing.
- One immutable `UiState` per screen.
- `StateFlow` collected with lifecycle awareness.
- String resources only.
- Accessible labels, 48dp touch targets, and visible focus states.
- No dynamic colour by default.

### Auth flow

- Login and registration call the API and send
  `X-Refresh-Token-Transport: body`.
- Access and refresh tokens are encrypted with an AES-GCM key in Android
  Keystore before being written to DataStore.
- The bearer interceptor adds the access token to protected requests.
- The OkHttp authenticator refreshes once on `401`, updates both tokens, and
  retries the original request.
- A failed refresh clears the session and returns the app to signed-out state.
- The refresh client must not recurse through the authenticator.
- RFC 9457 problem details are mapped to typed, user-safe errors.
- Tokens, passwords, and authorization headers are never logged.
- TOTP challenges are surfaced as an explicit unsupported state until the TOTP
  overlay is installed.

### Build and release

- `debug`: `.debug` application ID suffix, cleartext allowed only for local
  development hosts, redacted HTTP logging.
- `staging`: `.staging` suffix, HTTPS API URL, R8 disabled unless configured.
- `release`: strict HTTPS, R8 and resource shrinking, no debug logging, mapping
  output retained, signing read from environment variables.
- The API base URL is a `BuildConfig` field.
- Generated `.gitignore` excludes local Android and signing files.
- The Gradle wrapper and version catalog are committed.
- A minimal `build-logic` included build provides application, library, Compose,
  Hilt, and Kotlin serialization convention plugins.

## Toolchain

Initial pinned set, validated by a toolchain smoke build before expanding the
template:

| Component | Version |
| --- | --- |
| JDK | 17 |
| Gradle | 9.9.2 |
| Android Gradle Plugin | 9.5.2 |
| Kotlin | 2.2.10 (AGP built-in) |
| KSP | 2.3.14 |
| Hilt | 2.60.1 |
| Compose BOM | 2026.11.01 |
| compileSdk / targetSdk | 37.1 / 37 |
| minSdk | 26 |

The smoke build confirmed this combination on the available SDK. Compose 1.13
and the current AndroidX libraries require compileSdk 37.1 and AGP 9; the base
uses AGP 9 built-in Kotlin rather than the legacy Kotlin Android plugin.

## Generated project contract

The Android stack exposes three variables:

- `application_id`: Android application ID, default `com.example.app`.
- `package_name`: Kotlin package and namespace, default `com.example.app`.
- `api_base_url`: API endpoint, default `http://10.0.2.2:8080`.

The Kotlin source path is rendered with the CLI `packagePath` helper:

```text
{{ packagePath .package_name }}
```

The generated app includes:

- Gradle wrapper and version catalog.
- `build-logic` convention plugins.
- Production modules and source trees.
- Adaptive launcher icon derived from Smoke Hex assets.
- `README.md`, `AGENTS.md`, `CLAUDE.md`, `CONTEXT.md`, `DESIGN.md`, and `docs/`.
- Project-scoped agent skills and `skills-lock.json`.
- Tests, lint configuration, and a Makefile.

## CLI integration

- Add `replace` and `packagePath` to the render function map.
- Add mobile selection to standalone projects through the existing kind/stack
  path.
- Add `--mobile android/compose` for fullstack projects.
- Fullstack mobile requires API `auth:jwt`.
- Fullstack target path is `apps/android`.
- Fullstack mobile variables are validated separately from desktop variables.
- Root workspace output gains mobile Makefile targets, README/context sections,
  and a conditional Android CI job.
- Generated root architecture output includes the Android runtime edge and
  target rows.
- Standalone next steps use Gradle directly; fullstack next steps use root Make
  targets.

## Registry integration

- Add `mobile` kind with capacity for future iOS support.
- Add `android/compose` as the default mobile stack.
- Add a `mobile` default preset.
- Add `overlays/workspace/fullstack/mobile`.
- Add `tests/verify-android.sh` for standalone and fullstack generation.
- Add a registry CI `verify-android` job with JDK 17 and Android SDK 37.1.

## Testing

- CLI unit tests for rendering, selection, resolution, architecture, and output.
- Android JVM unit tests for the auth ViewModel and RFC 9457 mapper in the base.
- MockWebServer repository and refresh-rotation tests are follow-ups as the API
  surface grows.
- Robolectric Compose smoke tests are a follow-up overlay; previews and stateless
  screens keep the UI testable in the meantime.
- Generated app quality gate: `test`, `lint`, `assembleDebug`, and
  `assembleRelease`.
- No emulator or device required in CI.

## Delivery order

1. CLI rendering and mobile selection.
2. Toolchain smoke app and Gradle skeleton.
3. Core modules and session layer.
4. Features and app shell.
5. Production variants, docs, and agent files.
6. Registry manifest, template, workspace overlay, and CI.
7. Local standalone/fullstack matrix.
8. CLI release, registry CI pin update, registry release.

No release is published until the generated app and both repositories pass
verification.

## Out of scope

- Kotlin Multiplatform and Compose Multiplatform.
- Room, WorkManager, offline mutations, sync, and conflict handling.
- Google OAuth and TOTP in the base.
- Crash reporting, analytics, and remote configuration.
- Play publishing, signing key provisioning, and staged rollout automation.
- Screenshot testing, macrobenchmarks, and baseline profiles.
- OpenAPI-generated Kotlin clients.
