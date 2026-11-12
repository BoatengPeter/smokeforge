// Package project reads and writes smokeforge project state and lockfiles.
package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// StateFile is the project state file name.
	StateFile = ".smokeforge.json"
	// LockFile is the project lockfile name.
	LockFile = ".smokeforge.lock.json"
)

// State records the choices and generated artifacts for a project.
type State struct {
	SchemaVersion int               `json:"schemaVersion"`
	Name          string            `json:"name"`
	Kind          string            `json:"kind"`
	CreatedAt     string            `json:"createdAt"`
	Registry      RegistryRef       `json:"registry"`
	Variables     map[string]string `json:"variables"`
	Targets       []Target          `json:"targets"`
	RootFeatures  []string          `json:"rootFeatures"`
	Applied       []AppliedOverlay  `json:"applied"`
}

// RegistryRef identifies the registry revision a project was generated from.
type RegistryRef struct {
	Source string `json:"source"`
	Tag    string `json:"tag"`
	Commit string `json:"commit"`
}

// Target records one generated target and its selected features.
type Target struct {
	Path     string   `json:"path"`
	Stack    string   `json:"stack"`
	Features []string `json:"features"`
}

// AppliedOverlay records an overlay's effects for drift detection and removal.
type AppliedOverlay struct {
	Overlay     string            `json:"overlay"`
	Target      string            `json:"target"`
	RegistryTag string            `json:"registryTag"`
	Files       map[string]string `json:"files"`
	Markers     []string          `json:"markers"`
}

// Lock pins the registry and template checksums for reproducibility.
type Lock struct {
	LockVersion int               `json:"lockVersion"`
	Registry    RegistryLock      `json:"registry"`
	Templates   map[string]string `json:"templates"`
	Overlays    map[string]string `json:"overlays"`
}

// RegistryLock pins the registry source, tag, commit, and archive checksum.
type RegistryLock struct {
	Source        string `json:"source"`
	Tag           string `json:"tag"`
	Commit        string `json:"commit"`
	ArchiveSHA256 string `json:"archiveSHA256"`
}

// WriteState writes both project state files atomically, one file at a time.
func WriteState(dir string, state *State, lock *Lock) error {
	if err := state.Validate(); err != nil {
		return err
	}

	if err := lock.Validate(); err != nil {
		return err
	}

	if err := writeJSON(filepath.Join(dir, StateFile), state); err != nil {
		return fmt.Errorf("write project state: %w", err)
	}

	if err := writeJSON(filepath.Join(dir, LockFile), lock); err != nil {
		return fmt.Errorf("write project lock: %w", err)
	}

	return nil
}

// LoadState reads and validates the project state file.
func LoadState(dir string) (*State, error) {
	var state State
	if err := readJSON(filepath.Join(dir, StateFile), &state); err != nil {
		return nil, fmt.Errorf("load project state: %w", err)
	}

	if err := state.Validate(); err != nil {
		return nil, err
	}

	return &state, nil
}

// LoadLock reads and validates the project lockfile.
func LoadLock(dir string) (*Lock, error) {
	var lock Lock
	if err := readJSON(filepath.Join(dir, LockFile), &lock); err != nil {
		return nil, fmt.Errorf("load project lock: %w", err)
	}

	if err := lock.Validate(); err != nil {
		return nil, err
	}

	return &lock, nil
}

// Validate checks the state schema and required fields.
func (s *State) Validate() error {
	if s.SchemaVersion != 1 {
		return fmt.Errorf("unsupported state schemaVersion %d: expected 1", s.SchemaVersion)
	}

	if s.Name == "" {
		return errors.New("state name must not be empty")
	}

	if s.Kind == "" {
		return errors.New("state kind must not be empty")
	}

	if len(s.Targets) == 0 {
		return errors.New("state targets must not be empty")
	}

	for _, target := range s.Targets {
		if target.Path == "" || target.Stack == "" {
			return errors.New("state target path and stack must not be empty")
		}
	}

	return nil
}

// Validate checks the lock schema and required fields.
func (l *Lock) Validate() error {
	if l.LockVersion != 1 {
		return fmt.Errorf("unsupported lockVersion %d: expected 1", l.LockVersion)
	}

	if l.Registry.Tag == "" {
		return errors.New("lock registry tag must not be empty")
	}

	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %q: %w", path, err)
	}

	data = append(data, '\n')

	dir := filepath.Dir(path)

	temp, err := os.CreateTemp(dir, ".smokeforge-*")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", path, err)
	}

	tempName := temp.Name()

	cleanup := func() {
		_ = os.Remove(tempName)
	}

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()

		cleanup()

		return fmt.Errorf("write temp file for %q: %w", path, err)
	}

	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()

		cleanup()

		return fmt.Errorf("chmod temp file for %q: %w", path, err)
	}

	if err := temp.Close(); err != nil {
		cleanup()

		return fmt.Errorf("close temp file for %q: %w", path, err)
	}

	if err := os.Rename(tempName, path); err != nil {
		cleanup()

		return fmt.Errorf("commit %q: %w", path, err)
	}

	return nil
}

func readJSON(path string, value any) error {
	//nolint:gosec // path is a known project state file under a validated project directory.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("parse %q: %w", path, err)
	}

	return nil
}
