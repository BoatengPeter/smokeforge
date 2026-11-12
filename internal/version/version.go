// Package version provides build-time version metadata for smokeforge.
package version

// Build metadata. Release builds set these with -ldflags.
var (
	// Version is the semantic version or "dev" for local builds.
	Version = "dev"
	// Commit is the git commit the binary was built from.
	Commit = "none"
	// Date is the build date in RFC 3339 format.
	Date = "unknown"
)

// String returns the human-readable version, commit, and build date.
func String() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
