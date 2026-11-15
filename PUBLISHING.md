# Publishing smokeforge

Manual release instructions for maintainers. Follow these steps when you want to
publish a version. For background and rationale, see
[docs/development/release.md](docs/development/release.md).

**Current versions:** CLI `v0.2.0`, registry `v0.4.0`.

**Current policy:** pre-1.0 releases use `v0.x.y`. Reserve `v1.0.0` until
`install`, Homebrew, and final verification are complete.
The CLI release workflow creates a **draft** release; review it before publishing.
The registry workflow publishes its release immediately when a registry tag is
pushed.

---

## 1. Prerequisites

- Push access to `github.com/BoatengPeter/smokeforge`.
- Go 1.26.11+ and git.
- `gh` authenticated (`gh auth status`) with repository write access.
- goreleaser v2 for local builds (`goreleaser --version`). Install from
  https://goreleaser.com/install/ or use the release workflow.
- A clean `main` branch with green CI.

Optional environment for a manual goreleaser publish:

```bash
export GITHUB_TOKEN="$(gh auth token)"
```

The token must be allowed to create releases and upload assets. Prefer a
short-lived, minimally scoped token.

## 2. Choose the version

Pick the next semantic version:

- New features or milestone work: `v0.<minor>.0`
- Fixes/backports: `v0.<minor>.<patch>`
- `v1.0.0`: only when the full v1 scope is complete.

Set it once for the session:

```bash
VERSION=v0.1.9
```

## 3. Pre-flight verification

Run from a clean checkout of `main`:

```bash
git switch main
git pull --ff-only

go test ./...
go test -tags=integration ./...
go test -race ./...
go build ./...
golangci-lint run ./...
make vulncheck
goreleaser check
```

Confirm CI is green on Linux, macOS, and Windows, including the cross-build and
security jobs.

Optional local snapshot (no publishing):

```bash
make release-snapshot
# or: goreleaser release --snapshot --clean
```

Confirm the Linux archive reports injected metadata:

```bash
tar -xzf dist/smokeforge_*_linux_amd64.tar.gz -C /tmp/smokeforge
/tmp/smokeforge/smokeforge version
```

Stop if any command fails.

## 4. Update the changelog

Edit [CHANGELOG.md](CHANGELOG.md):

1. Move the relevant `## [Unreleased]` entries into a new section:

   ```markdown
   ## [0.1.0] - 2026-11-12
   ```

2. Leave a fresh empty `## [Unreleased]` section.
3. Do not claim features that are not shipped (`install`, fullstack, Homebrew,
   v1.0.0).

Commit:

```bash
git add CHANGELOG.md
git commit -m "docs: prepare v${VERSION#v} release notes"
git push origin main
```

Wait for CI on `main` to pass.

## 5. Publish via the release workflow (recommended)

Create and push an annotated tag:

```bash
git tag -a "$VERSION" -m "smokeforge $VERSION"
git push origin "$VERSION"
```

Watch the workflow:

```bash
gh run list --workflow=release.yml --limit 5
gh run watch
```

The workflow runs unit and integration tests, builds Linux/macOS/Windows
archives for amd64/arm64, writes `checksums.txt`, and creates a **draft**
GitHub release.

Review the draft:

```bash
gh release view "$VERSION" --json name,tagName,isDraft,assets
```

Verify at least:

```text
smokeforge_<version>_linux_amd64.tar.gz
smokeforge_<version>_linux_arm64.tar.gz
smokeforge_<version>_darwin_amd64.tar.gz
smokeforge_<version>_darwin_arm64.tar.gz
smokeforge_<version>_windows_amd64.zip
smokeforge_<version>_windows_arm64.zip
checksums.txt
```

Publish the draft:

```bash
gh release edit "$VERSION" --draft=false --latest
```

Verify the release and the updater:

```bash
gh release view "$VERSION" --json isDraft,assets
smokeforge update --check
smokeforge update --version "$VERSION" --force
```

## 6. Manual publish without the workflow

Use this if Actions is unavailable or you need an offline/reproducible build.

Build and publish with goreleaser:

```bash
git switch main
git pull --ff-only

GITHUB_TOKEN="$(gh auth token)" goreleaser release --clean
```

This creates the draft release and uploads artifacts. Review and publish it with
the same `gh release edit` command as above.

If goreleaser cannot publish, create a draft release and upload `dist/` manually:

```bash
goreleaser release --snapshot --clean

gh release create "$VERSION" dist/* \
  --verify-tag \
  --draft \
  --title "smokeforge $VERSION" \
  --notes-file CHANGELOG.md
```

Then review and publish in the GitHub UI. Do not upload `dist/config.yaml` or
`dist/metadata.json`; upload only binaries, archives, and `checksums.txt` if
using the UI.

## 7. Verify the published release

Checksums:

```bash
cd /tmp
gh release download "$VERSION" --pattern 'smokeforge_*_linux_amd64.tar.gz'
gh release download "$VERSION" --pattern checksums.txt
sha256sum --check checksums.txt --ignore-missing
```

`go install`:

```bash
GOBIN=/tmp/smokeforge-go-install \
  go install github.com/BoatengPeter/smokeforge/cmd/smokeforge@"$VERSION"

/tmp/smokeforge-go-install/smokeforge version
```

The version must match the tag; commit and date must be present.

Prebuilt archive:

```bash
tar -xzf smokeforge_"${VERSION#v}"_linux_amd64.tar.gz -C /tmp/smokeforge-release
/tmp/smokeforge-release/smokeforge version
```

## 8. Homebrew (when the tap exists)

Homebrew is configured but commented out in `.goreleaser.yaml`.

1. Create `github.com/BoatengPeter/homebrew-tap`.
2. Create a token with write access to the tap and add it as the
   `HOMEBREW_TAP_GITHUB_TOKEN` repository secret.
3. Uncomment `brews` in `.goreleaser.yaml` and keep:

   ```yaml
   token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
   ```

4. Add the token to the release workflow environment:

   ```yaml
   HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
   ```

5. Commit and cut a patch release.
6. Verify:

   ```bash
   brew install BoatengPeter/tap/smokeforge
   smokeforge version
   ```

## 9. Registry release

The registry is versioned separately from the CLI. Current latest is
`v0.5.2`.

### Recommended: tag-driven workflow

In the registry repository:

```bash
git switch main
git pull --ff-only
git tag -a v0.5.3 -m "smokeforge-registry v0.5.3"
git push origin v0.5.3
```

The registry release workflow packages `registry.tar.gz` with a top-level
`registry/` directory, writes `registry.sha256`, creates the GitHub release, and
marks it latest. Watch it with:

```bash
gh run list --repo BoatengPeter/smokeforge-registry --workflow=release.yml
gh release view v0.5.3 --repo BoatengPeter/smokeforge-registry --json isDraft,assets
```

### Manual fallback

```bash
rm -rf dist
mkdir -p dist/registry
rsync -a --exclude .git --exclude dist ./ dist/registry/
tar -czf dist/registry.tar.gz -C dist registry
(cd dist && sha256sum registry.tar.gz > registry.sha256)

gh release create v0.5.3 \
  --repo BoatengPeter/smokeforge-registry \
  dist/registry.tar.gz \
  dist/registry.sha256 \
  --title "smokeforge-registry v0.5.3" \
  --notes "Registry release v0.5.3" \
  --latest
```

The URL layout is:

```text
<source>/download/<tag>/registry.tar.gz
<source>/download/<tag>/registry.sha256
<source>/latest/download/registry.tar.gz
```

Smoke-test:

```bash
smokeforge registry update
smokeforge registry info
smokeforge list stacks --offline
```

## 10. Post-release

- Confirm the release is not a draft and is marked latest.
- Confirm checksums and both `go install` and archive installs work.
- Announce only the capabilities that shipped.
- Open follow-up issues for deferred items (`install`, fullstack, Homebrew tap,
  mutation dry-runs).
- If a release is broken:
  - If still a draft, delete it: `gh release delete "$VERSION" --yes`.
  - If published, do not delete the tag. Mark the release as a pre-release or
    publish a patch release with the fix, and note the issue in the changelog.

## Release checklist

- [ ] Version chosen and follows the pre-1.0 policy.
- [ ] `main` clean and CI green.
- [ ] Unit, integration, race, lint, build, and `goreleaser check` pass.
- [ ] Snapshot artifacts and version metadata verified.
- [ ] CHANGELOG updated; deferred items not claimed.
- [ ] Tag pushed or manual draft created.
- [ ] Draft assets and `checksums.txt` reviewed.
- [ ] `go install` and archive verification pass.
- [ ] Homebrew configured only after the tap exists.
- [ ] Registry tag or manual assets published; `registry update` and `list stacks` pass.
- [ ] License chosen before a public v1 release.

## Related documents

- [Release guide](docs/development/release.md)
- [Changelog](CHANGELOG.md)
- [Roadmap](docs/development/roadmap.md)
- [Testing](docs/development/testing.md)
