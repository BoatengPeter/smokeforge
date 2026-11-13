package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestCacheFreshness(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte("name: test\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if !cacheFresh(dir, time.Hour) {
		t.Fatal("new cache entry must be fresh")
	}

	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("age manifest: %v", err)
	}

	if cacheFresh(dir, time.Hour) {
		t.Fatal("old cache entry must be stale")
	}
}

func TestCacheVersionMatches(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if cacheVersionMatches(dir) {
		t.Fatal("missing version marker must not match")
	}

	if err := writeCacheVersion(dir); err != nil {
		t.Fatalf("write version: %v", err)
	}

	if !cacheVersionMatches(dir) {
		t.Fatal("written version marker must match")
	}
}

func TestCacheUsable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte("name: test\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if cacheUsable(Options{Tag: "latest"}, dir) {
		t.Fatal("latest cache without a version marker must be stale")
	}

	if err := writeCacheVersion(dir); err != nil {
		t.Fatalf("write version: %v", err)
	}

	if !cacheUsable(Options{Tag: "latest"}, dir) {
		t.Fatal("latest cache with matching version must be usable")
	}

	if !cacheUsable(Options{Tag: "latest", Offline: true}, t.TempDir()) {
		t.Fatal("offline mode must allow an existing cache entry")
	}

	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("age manifest: %v", err)
	}

	if cacheUsable(Options{Tag: "latest"}, dir) {
		t.Fatal("old latest cache must be stale")
	}

	if !cacheUsable(Options{Tag: "v1.0.0"}, dir) {
		t.Fatal("pinned tags must keep using their cache")
	}
}
