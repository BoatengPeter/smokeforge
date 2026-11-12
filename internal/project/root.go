package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotProject indicates no smokeforge project state file was found.
var ErrNotProject = errors.New("not inside a smokeforge project")

// FindRoot walks up from start until it finds a smokeforge project state file.
func FindRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve start directory: %w", err)
	}

	for {
		_, statErr := os.Stat(filepath.Join(dir, StateFile))
		if statErr == nil {
			return dir, nil
		}

		if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("check for project state in %q: %w", dir, statErr)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotProject
		}

		dir = parent
	}
}

func safeTrackedPath(value string) (string, error) {
	local := filepath.FromSlash(value)
	if value == "" || filepath.IsAbs(local) || !filepath.IsLocal(local) {
		return "", fmt.Errorf("path %q must be a safe relative path", value)
	}

	return filepath.ToSlash(filepath.Clean(local)), nil
}
