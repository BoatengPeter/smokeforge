package registry

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultCacheDir(t *testing.T) {
	t.Parallel()

	dir, err := DefaultCacheDir()
	if err != nil {
		t.Fatalf("cache dir: %v", err)
	}

	if !strings.Contains(dir, "smokeforge") {
		t.Fatalf("cache dir = %q, want a smokeforge path", dir)
	}

	if !filepath.IsAbs(dir) {
		t.Fatalf("cache dir = %q, want an absolute path", dir)
	}
}

func TestEntryDir(t *testing.T) {
	t.Parallel()

	cache := t.TempDir()
	first := entryDir(cache, "https://example.com/releases", "v1.0.0")
	second := entryDir(cache, "https://example.com/releases", "v1.0.0")
	otherTag := entryDir(cache, "https://example.com/releases", "v1.1.0")
	otherSource := entryDir(cache, "https://example.org/releases", "v1.0.0")

	if first != second {
		t.Fatalf("entry dir is not deterministic: %q != %q", first, second)
	}

	if first == otherTag {
		t.Fatal("different tags must use different cache entries")
	}

	if first == otherSource {
		t.Fatal("different sources must use different cache entries")
	}

	if !strings.HasPrefix(first, cache) {
		t.Fatalf("entry dir %q escapes cache %q", first, cache)
	}
}
