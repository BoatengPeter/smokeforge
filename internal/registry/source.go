package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
)

const DefaultURL = "https://github.com/BoatengPeter/smokeforge-registry/releases"

type Options struct {
	Source   string
	Tag      string
	CacheDir string
	Offline  bool
}

type Result struct {
	Manifest  *manifest.Manifest
	Source    string
	Tag       string
	Dir       string
	FromCache bool
}

func Load(ctx context.Context, opts Options) (*Result, error) {
	source := opts.Source
	if source == "" {
		source = DefaultURL
	}

	info, err := os.Stat(source)
	if err == nil && info.IsDir() {
		return loadLocal(source)
	}
	return loadRemote(ctx, opts)
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

func loadRemote(ctx context.Context, opts Options) (*Result, error) {
	return nil, fmt.Errorf("remote registry loading is not implemented yet")
}
