// Package version provides build-time version metadata for smokeforge.
package version

import (
	"runtime/debug"
	"strings"
)

// Build metadata. Release builds set these with -ldflags.
var (
	// Version is the semantic version or "dev" for local builds.
	Version = "dev"
	// Commit is the git commit the binary was built from.
	Commit = "none"
	// Date is the build date in RFC 3339 format.
	Date = "unknown"
)

// Info holds resolved version, commit, and build date metadata.
type Info struct {
	Version string
	Commit  string
	Date    string
}

// Current resolves build metadata from ldflags first, then from Go module build
// information. The fallback makes `go install ...@version` builds report the
// installed module version and VCS revision.
func Current() Info {
	current := Info{Version: Version, Commit: Commit, Date: Date}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return current
	}

	return mergeBuildInfo(current, buildInfo)
}

// String returns the human-readable version, commit, and build date.
func String() string {
	return format(Current())
}

func format(info Info) string {
	details := make([]string, 0, 2)
	if knownMetadata(info.Commit) {
		details = append(details, info.Commit)
	}

	if knownMetadata(info.Date) {
		details = append(details, info.Date)
	}

	if len(details) == 0 {
		return info.Version
	}

	return info.Version + " (" + strings.Join(details, ", ") + ")"
}

func knownMetadata(value string) bool {
	return value != "" && value != "none" && value != "unknown"
}

func mergeBuildInfo(current Info, buildInfo *debug.BuildInfo) Info {
	if isDevelopmentVersion(current.Version) {
		moduleVersion := buildInfo.Main.Version
		if moduleVersion != "" && moduleVersion != "(devel)" {
			current.Version = moduleVersion
		}
	}

	if current.Commit == "" || current.Commit == "none" {
		if revision := buildSetting(buildInfo, "vcs.revision"); revision != "" {
			current.Commit = revision
		}
	}

	if current.Date == "" || current.Date == "unknown" {
		if buildDate := buildSetting(buildInfo, "vcs.time"); buildDate != "" {
			current.Date = buildDate
		}
	}

	return current
}

func isDevelopmentVersion(value string) bool {
	return value == "" || value == "dev"
}

func buildSetting(buildInfo *debug.BuildInfo, key string) string {
	for _, setting := range buildInfo.Settings {
		if setting.Key == key {
			return setting.Value
		}
	}

	return ""
}
