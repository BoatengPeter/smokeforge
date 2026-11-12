// Package manifest defines the smokeforge template registry schema and its
// validation rules.
package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Manifest describes a template registry: its stack catalog, feature catalog,
// and per-kind selection limits.
type Manifest struct {
	SchemaVersion int             `yaml:"schemaVersion"`
	Name          string          `yaml:"name"`
	Kinds         map[string]Kind `yaml:"kinds"`
	Stacks        []Stack         `yaml:"stacks"`
	Features      []Feature       `yaml:"features"`
}

// Kind declares how many targets of a kind a project may select.
type Kind struct {
	Max int `yaml:"max"`
}

// Stack identifies a base template and the variables it requires.
type Stack struct {
	ID        string   `yaml:"id"`
	Kind      string   `yaml:"kind"`
	Name      string   `yaml:"name"`
	Template  string   `yaml:"template"`
	Variables []string `yaml:"variables"`
}

// Feature declares a composable capability, its variants, and its dependency
// and conflict rules.
type Feature struct {
	ID        string            `yaml:"id"`
	Name      string            `yaml:"name"`
	Kinds     []string          `yaml:"kinds"`
	Stacks    []string          `yaml:"stacks"`
	Variants  []Variant         `yaml:"variants"`
	Requires  []string          `yaml:"requires"`
	Conflicts []string          `yaml:"conflicts"`
	Overlays  map[string]string `yaml:"overlays"`
}

// Variant selects an implementation of a feature.
type Variant struct {
	ID       string            `yaml:"id"`
	Name     string            `yaml:"name"`
	Default  bool              `yaml:"default"`
	Overlays map[string]string `yaml:"overlays"`
}

// Parse decodes a single YAML manifest document and rejects unknown fields.
func Parse(data []byte) (*Manifest, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	var extra any

	err := dec.Decode(&extra)
	switch {
	case err == nil:
		return nil, errors.New("parse manifest: multiple YAML documents are not supported")
	case errors.Is(err, io.EOF):
		return &m, nil
	default:
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
}

// LoadFile reads and parses a manifest from path.
func LoadFile(path string) (*Manifest, error) {
	//nolint:gosec // path is chosen by the CLI or resolved from a verified cache entry.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}

	return Parse(data)
}
