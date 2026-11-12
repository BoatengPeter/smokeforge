package project

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

// Drift describes one tracked file whose current content no longer matches the
// hash recorded in project state.
type Drift struct {
	Path     string
	Expected string
	Actual   string
}

// CheckDrift compares every file recorded in state against the project tree.
// Missing files are reported with Actual "missing".
func CheckDrift(root string, state *State) ([]Drift, error) {
	if state == nil {
		return nil, errors.New("check drift: state is required")
	}

	drifts := make([]Drift, 0)

	for _, applied := range state.Applied {
		fileDrifts, err := checkFiles(root, applied.Files)
		if err != nil {
			return nil, err
		}

		drifts = append(drifts, fileDrifts...)
	}

	sort.Slice(drifts, func(left, right int) bool {
		return drifts[left].Path < drifts[right].Path
	})

	return drifts, nil
}

func checkFiles(root string, files map[string]string) ([]Drift, error) {
	drifts := make([]Drift, 0, len(files))

	for _, path := range slices.Sorted(maps.Keys(files)) {
		expected := files[path]

		clean, err := safeTrackedPath(path)
		if err != nil {
			return nil, fmt.Errorf("check drift: %w", err)
		}

		full := filepath.Join(root, filepath.FromSlash(clean))

		info, statErr := os.Stat(full)
		if errors.Is(statErr, os.ErrNotExist) {
			drifts = append(drifts, Drift{Path: clean, Expected: expected, Actual: "missing"})

			continue
		}

		if statErr != nil {
			return nil, fmt.Errorf("check drift for %q: %w", path, statErr)
		}

		if !info.Mode().IsRegular() {
			drifts = append(drifts, Drift{Path: clean, Expected: expected, Actual: "not a regular file"})

			continue
		}

		actual, err := HashFile(full)
		if err != nil {
			return nil, err
		}

		if actual != expected {
			drifts = append(drifts, Drift{Path: clean, Expected: expected, Actual: actual})
		}
	}

	return drifts, nil
}

// Rehash updates every recorded file hash from the current project tree and
// drops files that no longer exist.
func Rehash(root string, state *State) error {
	if state == nil {
		return errors.New("rehash: state is required")
	}

	for index := range state.Applied {
		files := state.Applied[index].Files

		for path := range files {
			clean, err := safeTrackedPath(path)
			if err != nil {
				return fmt.Errorf("rehash: %w", err)
			}

			full := filepath.Join(root, filepath.FromSlash(clean))

			info, statErr := os.Stat(full)
			if errors.Is(statErr, os.ErrNotExist) {
				delete(files, path)

				continue
			}

			if statErr != nil {
				return fmt.Errorf("rehash %q: %w", path, statErr)
			}

			if !info.Mode().IsRegular() {
				return fmt.Errorf("rehash %q: not a regular file", path)
			}

			hash, err := HashFile(full)
			if err != nil {
				return err
			}

			files[path] = hash
		}
	}

	return nil
}
