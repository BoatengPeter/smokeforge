# Security policy

## Supported versions

smokeforge is pre-1.0. Security fixes are applied to the latest released version
and `main`.

## Reporting a vulnerability

Please do not open a public issue for a suspected vulnerability.

Use GitHub private vulnerability reporting if it is enabled for this repository
(Security tab → Report a vulnerability). If it is not enabled, contact the
maintainer through the GitHub profile linked in the repository metadata and
include:

- affected version/commit
- a minimal reproduction
- impact assessment
- any suggested fix

Reports are handled on a best-effort basis.

## Security model

- Templates and registry content are **declarative data**. The CLI never executes
  template-provided shell commands, scripts, or plugins during generation.
- Remote registries require HTTPS (HTTP is allowed only for loopback hosts) and
  are SHA-256 verified before extraction.
- Redirects are validated; redirects to insecure or credentialed URLs are
  rejected.
- Archives are extracted with `os.Root`; traversal, links, devices, and
  oversized entries are rejected.
- Manifest, template, overlay, and rendered paths are confined to their roots.
- Secrets are generated with `crypto/rand`, written only to `.env` (`0600`), and
  excluded from project state hashes.
- No telemetry is collected.
- Release artifacts include SHA-256 checksums and are published from reviewed
  draft releases.

See [docs/development/security.md](docs/development/security.md) for the full
threat model and controls, and
[docs/development/cross-platform.md](docs/development/cross-platform.md) for the
supported platform matrix.

## Known limitations

- Registry/template signing and SBOM generation are not implemented.
- Local registry directories are trusted inputs.
- A license has not been selected yet; do not redistribute publicly until the
  project license is added.
- `smokeforge install` is not implemented. When added, it will use built-in
  per-stack adapters rather than arbitrary registry-provided commands.
