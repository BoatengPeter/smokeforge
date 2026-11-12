# Release guide

**Status:** release engineering is prepared. Publishing a release, creating the
Homebrew tap, and tagging `v1.0.0` are pending.

## Versioning policy

- Pre-1.0 releases use `v0.x.y`.
- `v1.0.0` is reserved for the completed v1 scope: M3 registry content, `install`,
  fullstack composition, and release verification.
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
   git tag -a v0.1.0 -m "smokeforge v0.1.0"
   git push origin v0.1.0
   ```

5. The `release` workflow runs tests, builds six platform archives, generates
   `checksums.txt`, and creates a **draft** GitHub release.
6. Review the draft: artifacts, checksums, changelog, and version metadata.
7. Publish the release.
8. Verify installation:

   ```bash
   go install github.com/BoatengPeter/smokeforge/cmd/smokeforge@v0.1.0
   smokeforge version
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

## Registry release (deferred with M3)

The template registry is a separate repository. When M3 is unblocked:

1. Tag the registry (`v1.0.0`).
2. Package the tree so the archive contains a top-level `registry/` directory:

   ```bash
   tar -czf registry.tar.gz registry/
   sha256sum registry.tar.gz > registry.sha256
   ```

3. Attach `registry.tar.gz` and `registry.sha256` to the registry GitHub release.
4. The resulting URL layout must be:

   ```text
   <source>/download/<tag>/registry.tar.gz
   <source>/download/<tag>/registry.sha256
   ```

5. Smoke-test fetching and caching:

   ```bash
   smokeforge --tag v1.0.0 registry update
   smokeforge --tag v1.0.0 list stacks --offline
   ```

## Pre-release checklist

- [ ] Unit, integration, race, lint, and build pass on `main`.
- [ ] CHANGELOG is current; no unverified claims.
- [ ] Snapshot artifacts and checksums verified.
- [ ] Version metadata appears in `smokeforge version`.
- [ ] Draft release reviewed before publishing.
- [ ] License chosen and added before a public v1 release.
- [ ] Registry assets published when M3 is complete.

## Related documents

- [Setup](setup.md)
- [Testing](testing.md)
- [Roadmap](roadmap.md)
- [README installation section](../../README.md#installation)
