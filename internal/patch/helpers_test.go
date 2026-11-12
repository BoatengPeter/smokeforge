package patch

import (
	"os"
	"testing"
)

func mustMap(t *testing.T, value any, key string) map[string]any {
	t.Helper()

	mapping, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%q is %T, want object", key, value)
	}

	return mapping
}

func mustSlice(t *testing.T, value any, key string) []any {
	t.Helper()

	items, ok := value.([]any)
	if !ok {
		t.Fatalf("%q is %T, want array", key, value)
	}

	return items
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	//nolint:gosec // test reads a known file under a temporary directory.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data
}
