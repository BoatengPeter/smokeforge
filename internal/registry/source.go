// Package registry loads, verifies, and caches smokeforge template registries.
package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
)

// DefaultURL is the release base URL of the official smokeforge registry.
const DefaultURL = "https://github.com/BoatengPeter/smokeforge-registry/releases"

// Options controls where a registry is loaded from and how it is cached.
type Options struct {
	// Source is a local directory or an HTTP(S) release base URL. An empty
	// value uses DefaultURL.
	Source string
	// Tag is the registry release tag. Local sources ignore it.
	Tag string
	// CacheDir overrides the default user cache directory.
	CacheDir string
	// Offline forbids network access and requires a warm cache.
	Offline bool
}

// Result is a validated registry manifest together with its origin.
type Result struct {
	Manifest  *manifest.Manifest
	Source    string
	Tag       string
	Dir       string
	FromCache bool
}

// Load returns a validated registry manifest from a local directory or a
// remote release archive, using and populating the cache as needed.
func Load(ctx context.Context, opts Options) (*Result, error) {
	source := opts.Source
	if source == "" {
		source = DefaultURL
	}

	if isLocalSource(source) {
		return loadLocal(source)
	}

	opts.Source = source

	return loadRemote(ctx, opts)
}

func isLocalSource(source string) bool {
	info, err := os.Stat(source)
	if err == nil {
		return info.IsDir()
	}

	return !strings.Contains(source, "://")
}

func loadLocal(dir string) (*Result, error) {
	path := filepath.Join(dir, "manifest.yaml")

	m, err := manifest.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load local registry %q: %w", dir, err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate local registry %q: %w", dir, err)
	}

	return &Result{
		Manifest: m,
		Source:   dir,
		Tag:      "local",
		Dir:      dir,
	}, nil
}
