# Security

**Status:** controls implemented for M1/M2/M4/M5 release engineering. Signing and
SBOMs are future work.

## Trust boundaries

| Input | Trust | Controls |
| --- | --- | --- |
| Registry URL | Untrusted | HTTPS required (HTTP only for loopback), host/userinfo validation, redirect validation |
| Registry archive | Untrusted | SHA-256 checksum, size limits, safe `os.Root` extraction |
| Manifest / template / overlay | Untrusted data | Strict schemas, unknown fields rejected, relative path validation, no execution |
| Local registry directory | Trusted developer input | Symlink/non-regular template files rejected; paths confined |
| Generated project | Trusted after generation | Atomic staging, drift hashes, backups, restore on failure |
| Release artifacts | Published output | SHA-256 checksums, draft review, CI security scans |

## Registry fetching

- Only `https` is accepted for remote sources.
- `http` is accepted only for loopback hosts (`localhost`, `127.0.0.0/8`, `::1`)
  to support tests and local development.
- URLs with credentials (`user:pass@host`) are rejected.
- URLs without a host are rejected.
- Redirects are validated with the same rules; more than 10 redirects abort.
- Checksum files are limited to 4 KiB; archives are limited to 512 MiB
  compressed.
- Downloads are written to a private temp file, verified, then extracted and
  atomically renamed into the cache.

## Archive extraction

- Extraction uses `os.Root`, so entries cannot escape the destination even via
  symlinks.
- Absolute paths and `..` traversal are rejected.
- Symlinks, hardlinks, devices, and unsupported entry types are rejected.
- Limits: 10,000 entries, 100 MiB per file, 512 MiB uncompressed total.
- File modes are normalized; executable bits are preserved for generated files.
- The context is checked between entries so cancellation is prompt.

## Templates and patches

- Templates cannot execute commands. The CLI performs every filesystem write.
- Strict templates fail on missing variables (`missingkey=error`).
- Rendered paths, overlay destinations, insert targets, patch files, and
  dependency files must be clean relative paths.
- Project writes use `os.Root`; overlay source reads are confined with safe
  joins.
- Anchor insertions are wrapped in markers and must match exactly once.
- JSON/YAML/TOML patches are re-parsed after edits.
- Dependency adapters use standard manifest parsers.
- Template directories reject symlinks and non-regular files.

## Secrets and state

- Secrets use `crypto/rand`, are written only to `.env` (`0600`), and are
  gitignored.
- `.env.example` contains documented placeholders.
- Project state and lockfiles do not contain secret values.
- `.env` is excluded from drift hashes because it is expected to be edited
  locally.
- State/lock writes use temp-file + rename.
- Backups live under `.smokeforge/backups/` and are gitignored.

## Lifecycle safety

- Drift is detected by comparing recorded SHA-256 hashes before mutation.
- Adds refuse on drift unless `--force`; removals always refuse.
- Removal supports only added files, marked insertions, and env keys. Structured
  config and dependency patches are refused until inverse records exist.
- Every mutation snapshots impacted files and state/lock first and restores on
  failure.

## Release and supply chain

- `golangci-lint` runs `gosec`, `bodyclose`, `noctx`, `errcheck`, `staticcheck`,
  and related security linters.
- CI runs `govulncheck` on every push and pull request.
- CI tests on Linux, macOS, and Windows and cross-compiles all six release
  targets.
- goreleaser publishes SHA-256 checksums, archives, and draft releases.
- Release tokens should be minimally scoped; Homebrew publishing requires a
  separate tap token.
- Review draft release assets before publishing.

## Verification commands

```bash
go test ./...
go test -tags=integration ./...
go test -race ./...
golangci-lint run ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
goreleaser check
```

## Known gaps

- No registry/template signing or transparency log.
- No SBOM or binary signing yet.
- Local registry directories are trusted.
- Native toolchain commands for `smokeforge install` are not implemented.
- License pending; see [SECURITY.md](../../SECURITY.md).

## Reporting

See [SECURITY.md](../../SECURITY.md).
