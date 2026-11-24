package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandsReportTooOldCLI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	manifest := `schemaVersion: 1
minCliVersion: 9.0.0
name: future
someFutureField: true
kinds:
  backend: {max: 1}
stacks: []
features: []
`
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"list", "stacks", "--registry", dir},
		{"registry", "update", "--registry", dir},
	} {
		var out, errOut bytes.Buffer

		cmd := newRootCmd(&out, &errOut, "0.3.7")
		cmd.SetContext(t.Context())
		cmd.SetArgs(args)

		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "smokeforge update") {
			t.Fatalf("%v: error = %v, want upgrade message", args, err)
		}
	}
}
