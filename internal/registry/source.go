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
	// CLIVersion is the running smokeforge version, checked against the
	// registry's minCliVersion. Empty or non-semver (development) versions
	// skip the check.
	CLIVersion string
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
		return loadLocal(source, opts.CLIVersion)
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

func loadLocal(dir, cliVersion string) (*Result, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve local registry %q: %w", dir, err)
	}

	dir = absolute

	path := filepath.Join(dir, "manifest.yaml")

	m, err := loadManifest(path, cliVersion)
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

// loadManifest checks minCliVersion before the strict parse, so an older CLI
// reports "upgrade smokeforge" rather than a schema error for fields it does
// not know yet.
func loadManifest(path, cliVersion string) (*manifest.Manifest, error) {
	//nolint:gosec // path is the manifest inside a local or verified cache registry.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}

	if err := manifest.CheckCompatibility(data, cliVersion); err != nil {
		return nil, err
	}

	m, err := manifest.Parse(data)
	if err != nil {
		return nil, err
	}

	return m, nil
}
