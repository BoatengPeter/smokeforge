# Release guide

**Status:** CLI and registry releases are active. Current CLI is `v0.2.4`;
current registry is `v0.9.2`. The Homebrew tap and `v1.0.0` remain pending.

## Versioning policy

- Pre-1.0 releases use `v0.x.y`.
- `v1.0.0` is reserved for the completed v1 scope: M3 registry content, `install`,
  and final verification.
- Build metadata is injected by goreleaser ldflags:
  - `internal/version.Version`
  - `internal/version.Commit`
  - `internal/version.Date`
- `go install ...@<version>` builds have no ldflags; `version.Current` falls back
  to Go module build info (`Main.Version`, `vcs.revision`, `vcs.time`).

## Local snapshot

Install goreleaser v2, then:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

or:

```bash
make release-check
make release-snapshot
```

Artifacts are written to `dist/` (gitignored):

```text
smokeforge_<version>_<os>_<arch>.tar.gz
smokeforge_<version>_windows_<arch>.zip
checksums.txt
```

Verify the Linux archive:

```bash
tar -xzf dist/smokeforge_*_linux_amd64.tar.gz -C /tmp/smokeforge
/tmp/smokeforge/smokeforge version
```

The version output must include the snapshot version, commit, and date.

## Cutting a release

1. Ensure `main` is green: unit, integration, race, lint, and build.
2. Move the `Unreleased` entries in [CHANGELOG.md](../../CHANGELOG.md) into the
   new version section with the release date.
3. Commit the changelog.
4. Create and push an annotated tag:

   ```bash
   git tag -a v0.2.4 -m "smokeforge v0.2.4"
   git push origin v0.2.4
   ```

5. The `release` workflow runs tests, builds six platform archives, generates
   `checksums.txt`, and creates a **draft** GitHub release.
6. Review the draft: artifacts, checksums, changelog, and version metadata.
7. Publish it:

   ```bash
   gh release edit v0.2.4 --draft=false --latest
   ```
8. Verify installation and self-update:

   ```bash
   go install github.com/BoatengPeter/smokeforge/cmd/smokeforge@v0.2.4
   smokeforge version
   smokeforge update --check
   ```

## Homebrew

Homebrew publishing is commented out in `.goreleaser.yaml` until a tap exists.

1. Create `github.com/BoatengPeter/homebrew-tap`.
2. Create a token with write access to the tap and add it as the
   `HOMEBREW_TAP_GITHUB_TOKEN` repository secret.
3. Uncomment the `brews` block, add:

   ```yaml
   token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
   ```

4. Add `HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}` to
   the release workflow environment.
5. Cut a release and verify `brew install BoatengPeter/tap/smokeforge`.

## Registry release

The template registry is released separately from the CLI. Current latest is
`v0.9.2`.

### Recommended: tag-driven workflow

```bash
git tag -a v0.8.1 -m "smokeforge-registry v0.8.1"
git push origin v0.8.1
```

The registry workflow publishes `registry.tar.gz` and `registry.sha256`
immediately. It packages the archive with a top-level `registry/` directory.

### Manual fallback

```bash
rm -rf dist
mkdir -p dist/registry
rsync -a --exclude .git --exclude dist ./ dist/registry/
tar -czf dist/registry.tar.gz -C dist registry
(cd dist && sha256sum registry.tar.gz > registry.sha256)

gh release create v0.8.1 \
  --repo BoatengPeter/smokeforge-registry \
  dist/registry.tar.gz dist/registry.sha256 \
  --title "smokeforge-registry v0.8.1" \
  --notes "Registry release v0.8.1" \
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

## Pre-release checklist

- [ ] Unit, integration, race, lint, build, cross-compile, and `govulncheck` pass.
- [ ] Security and cross-platform CI jobs are green.
- [ ] CHANGELOG is current; no unverified claims.
- [ ] Snapshot artifacts and checksums verified.
- [ ] Version metadata appears in `smokeforge version`.
- [ ] Draft release reviewed before publishing.
- [ ] License chosen and added before a public v1 release.
- [ ] Registry release published; `registry update` and `list stacks` pass.

## Related documents

- [Publishing checklist](../../PUBLISHING.md) — copy-paste maintainer steps
- [Setup](setup.md)
- [Testing](testing.md)
- [Security](security.md)
- [Cross-platform](cross-platform.md)
- [Roadmap](roadmap.md)
- [README installation section](../../README.md#installation)
