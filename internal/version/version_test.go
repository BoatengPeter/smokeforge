package version

import (
	"runtime/debug"
	"testing"
)

func TestFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info Info
		want string
	}{
		{
			name: "omits unknown metadata",
			info: Info{Version: "v0.1.0", Commit: "none", Date: "unknown"},
			want: "v0.1.0",
		},
		{
			name: "includes commit only",
			info: Info{Version: "v0.1.0", Commit: "abc123", Date: "unknown"},
			want: "v0.1.0 (abc123)",
		},
		{
			name: "includes commit and date",
			info: Info{Version: "v0.1.0", Commit: "abc123", Date: "2026-11-13T10:00:00Z"},
			want: "v0.1.0 (abc123, 2026-11-13T10:00:00Z)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := format(tt.info); got != tt.want {
				t.Fatalf("format = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeBuildInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current Info
		build   *debug.BuildInfo
		want    Info
	}{
		{
			name:    "fills development metadata",
			current: Info{Version: "dev", Commit: "none", Date: "unknown"},
			build: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.1.0"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.time", Value: "2026-11-12T12:00:00Z"},
				},
			},
			want: Info{Version: "v0.1.0", Commit: "abc123", Date: "2026-11-12T12:00:00Z"},
		},
		{
			name:    "keeps explicit ldflags values",
			current: Info{Version: "v9.9.9", Commit: "release-sha", Date: "2026-01-01T00:00:00Z"},
			build: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.1.0"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "build-sha"},
					{Key: "vcs.time", Value: "2026-11-12T12:00:00Z"},
				},
			},
			want: Info{Version: "v9.9.9", Commit: "release-sha", Date: "2026-01-01T00:00:00Z"},
		},
		{
			name:    "ignores devel module version",
			current: Info{Version: "dev", Commit: "none", Date: "unknown"},
			build: &debug.BuildInfo{
				Main: debug.Module{Version: "(devel)"},
			},
			want: Info{Version: "dev", Commit: "none", Date: "unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := mergeBuildInfo(tt.current, tt.build)
			if got != tt.want {
				t.Fatalf("info = %+v, want %+v", got, tt.want)
			}
		})
	}
}

//nolint:paralleltest // mutates package-level build metadata.
func TestCurrentPrefersExplicitMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	Version = "v1.2.3"
	Commit = "explicit-sha"
	Date = "2026-11-12T12:00:00Z"

	t.Cleanup(func() {
		Version = oldVersion
		Commit = oldCommit
		Date = oldDate
	})

	info := Current()
	if info.Version != "v1.2.3" || info.Commit != "explicit-sha" {
		t.Fatalf("info = %+v", info)
	}
}
