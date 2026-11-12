# Cross-platform compatibility

**Supported matrix:** Linux, macOS, and Windows on `amd64` and `arm64`.

## CI coverage

| Check | Linux | macOS | Windows |
| --- | --- | --- | --- |
| Unit tests | yes | yes | yes |
| Integration tests (`-tags=integration`) | yes | yes | yes |
| Race detector | yes | — | — |
| golangci-lint | yes | — | — |
| Cross-compile all six targets | yes | — | — |
| govulncheck | yes | — | — |

The release workflow builds all six targets; `goreleaser check` validates the
configuration.

## Code rules

- Use `filepath` for filesystem paths and `path` for slash-separated registry
  paths. Never assume `/` as a filesystem separator.
- Use `filepath.FromSlash`/`ToSlash` at boundaries.
- Use `os.Root` for project writes and archive extraction.
- Do not assume Unix permission bits control access; treat `0600`/`0755` as
  best-effort on Windows.
- Do not invoke a shell. External commands must use `exec.CommandContext` with
  separate arguments (`git init` today; install adapters later).
- Avoid platform-specific syscalls; `os.Root` and the standard library cover the
  supported targets.
- Registry paths use forward slashes and are validated before joining.
- Archive extraction rejects links; Windows ZIP releases are produced only by
  goreleaser, not extracted by the registry client.
- Timestamped backup names avoid `:` and other Windows-reserved characters.
- Templates are written with LF line endings; generated Go/Rust/Python/Node code
  accepts LF on all supported platforms.
- Tests must pass with `t.TempDir()` on Windows, including `t.Chdir` usage.

## Platform notes

### Windows

- `os.Chmod` largely ignores Unix permission bits; generated `.env` remains
  gitignored and state hashes exclude it.
- `os.Rename` over an existing empty directory is avoided: the engine removes
  the empty target before the final rename.
- Git may not be installed; use `--no-git` or install Git for Windows.
- ANSI styling is emitted only to terminals with color support; non-TTY output
  and `NO_COLOR` are plain text.

### macOS

- Apple Silicon and Intel are both built (`arm64`, `amd64`).
- `git` is available through the Xcode command line tools.

### Linux

- `amd64` and `arm64` archives are built.
- Permissions and `os.Root` behavior are exercised by the test suite.

## Release artifacts

| OS | Architectures | Format |
| --- | --- | --- |
| Linux | amd64, arm64 | `tar.gz` |
| macOS | amd64, arm64 | `tar.gz` |
| Windows | amd64, arm64 | `zip` |

All archives are listed in `checksums.txt`.

## Verification

Local cross-compile check:

```bash
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  goos="${target%/*}"
  goarch="${target#*/}"
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 \
    go build -trimpath -o "/tmp/smokeforge-${goos}-${goarch}" ./cmd/smokeforge
done
```

Run native tests on each platform when possible:

```bash
go test ./...
go test -tags=integration ./...
```

See [testing.md](testing.md), [release.md](release.md), and
[security.md](security.md).
