package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testManifest = `schemaVersion: 1
name: test
kinds:
  backend: {max: 1}
stacks:
  - {id: "go/chi", kind: backend, name: Go (chi), template: templates/go, variables: [module_path]}
features:
  - id: database
    name: Database
    kinds: [backend]
    variants:
      - {id: postgres, name: PostgreSQL}
`

func writeTestRegistry(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte(testManifest), 0o600); err != nil {
		t.Fatal(err)
	}

	return dir
}

func executeCommand(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	var out, errOut bytes.Buffer

	cmd := NewRootCmd(&out, &errOut)
	cmd.SetContext(t.Context())
	cmd.SetArgs(args)

	err = cmd.Execute()

	return out.String(), errOut.String(), err
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("stacks", func(t *testing.T) {
		t.Parallel()

		out, _, err := executeCommand(t, "list", "stacks", "--registry", writeTestRegistry(t))
		if err != nil {
			t.Fatalf("execute: %v", err)
		}

		for _, want := range []string{"go/chi", "Go (chi)"} {
			if !strings.Contains(out, want) {
				t.Fatalf("output %q missing %q", out, want)
			}
		}
	})

	t.Run("features", func(t *testing.T) {
		t.Parallel()

		out, _, err := executeCommand(t, "list", "features", "--registry", writeTestRegistry(t))
		if err != nil {
			t.Fatalf("execute: %v", err)
		}

		for _, want := range []string{"database", "postgres"} {
			if !strings.Contains(out, want) {
				t.Fatalf("output %q missing %q", out, want)
			}
		}
	})

	t.Run("rejects unknown target", func(t *testing.T) {
		t.Parallel()

		_, _, err := executeCommand(t, "list", "nope", "--registry", writeTestRegistry(t))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
