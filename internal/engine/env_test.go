package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BoatengPeter/smokeforge/internal/patch"
)

func TestWriteEnvFilesPreservesExistingValues(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("APP_URL=http://localhost:3000\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	existing := "APP_URL=https://example.com\nKEEP_ME=original\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = root.Close() })

	entries := []patch.Env{
		{Key: "APP_URL", Example: "http://localhost:3000"},
		{Key: "NEW_VALUE", Example: "new"},
		{Key: "SECRET_VALUE", Secret: true},
	}

	if err := writeEnvFiles(root, entries); err != nil {
		t.Fatal(err)
	}

	env := mustReadEnvFile(t, filepath.Join(dir, ".env"))
	example := mustReadEnvFile(t, filepath.Join(dir, ".env.example"))

	if !strings.Contains(env, "APP_URL=https://example.com\n") {
		t.Fatalf("existing value overwritten: %q", env)
	}

	if !strings.Contains(env, "KEEP_ME=original\n") {
		t.Fatalf("unrelated value removed: %q", env)
	}

	if !strings.Contains(env, "NEW_VALUE=new\n") {
		t.Fatalf("new value missing: %q", env)
	}

	if !strings.Contains(example, "NEW_VALUE=new\n") {
		t.Fatalf("example value missing: %q", example)
	}

	secret := envValue(t, env, "SECRET_VALUE")
	if len(secret) != 48 {
		t.Fatalf("secret length = %d, want 48", len(secret))
	}
}

func mustReadEnvFile(t *testing.T, path string) string {
	t.Helper()

	//nolint:gosec // The path is a test fixture under t.TempDir.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func envValue(t *testing.T, content, key string) string {
	t.Helper()

	for line := range strings.SplitSeq(content, "\n") {
		name, value, found := strings.Cut(line, "=")
		if found && name == key {
			return value
		}
	}

	t.Fatalf("missing env key %q", key)

	return ""
}
