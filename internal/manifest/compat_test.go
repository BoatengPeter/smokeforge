package manifest

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckCompatibility(t *testing.T) {
	t.Parallel()

	const requiresNewer = `schemaVersion: 1
minCliVersion: 0.4.0
name: future
futureField: this CLI does not know about it
`

	tests := []struct {
		name     string
		data     string
		current  string
		wantErr  bool
		tooOld   bool
		contains []string
	}{
		{name: "no requirement", data: "schemaVersion: 1\nname: x\n", current: "0.1.0"},
		{
			name:     "older CLI is told to upgrade even with unknown fields",
			data:     requiresNewer,
			current:  "0.3.7",
			wantErr:  true,
			tooOld:   true,
			contains: []string{"0.4.0", "0.3.7", "smokeforge update"},
		},
		{name: "v-prefixed older CLI", data: requiresNewer, current: "v0.3.7", wantErr: true, tooOld: true},
		{name: "exact version", data: requiresNewer, current: "0.4.0"},
		{name: "newer CLI", data: requiresNewer, current: "v0.5.1"},
		{name: "dev build skips the check", data: requiresNewer, current: "dev"},
		{name: "go install devel build skips the check", data: requiresNewer, current: "(devel)"},
		{name: "empty version skips the check", data: requiresNewer, current: ""},
		{
			name:     "malformed requirement",
			data:     "schemaVersion: 1\nminCliVersion: soon\n",
			current:  "0.3.7",
			wantErr:  true,
			contains: []string{"minCliVersion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckCompatibility([]byte(tt.data), tt.current)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("CheckCompatibility: %v", err)
				}

				return
			}

			if err == nil {
				t.Fatal("expected error")
			}

			if errors.Is(err, ErrCLITooOld) != tt.tooOld {
				t.Fatalf("errors.Is(err, ErrCLITooOld) = %v, want %v (err %v)", !tt.tooOld, tt.tooOld, err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q missing %q", err, want)
				}
			}
		})
	}
}

func TestParseAcceptsMinCLIVersion(t *testing.T) {
	t.Parallel()

	m, err := Parse([]byte("schemaVersion: 1\nminCliVersion: 0.4.0\nname: x\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if m.MinCLIVersion != "0.4.0" {
		t.Fatalf("MinCLIVersion = %q", m.MinCLIVersion)
	}
}

func TestValidateRejectsMalformedMinCLIVersion(t *testing.T) {
	t.Parallel()

	m := validManifest()
	m.MinCLIVersion = "next"

	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "minCliVersion") {
		t.Fatalf("error = %v, want minCliVersion error", err)
	}

	m.MinCLIVersion = "v0.4.0"
	if err := m.Validate(); err != nil {
		t.Fatalf("validate v-prefixed version: %v", err)
	}
}
