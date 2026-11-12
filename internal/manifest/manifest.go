package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	SchemaVersion int             `yaml:"schemaVersion"`
	Name          string          `yaml:"name"`
	Kinds         map[string]Kind `yaml:"kinds"`
	Stacks        []Stack         `yaml:"stacks"`
	Features      []Feature       `yaml:"features"`
}

type Kind struct {
	Max int `yaml:"max"`
}

type Stack struct {
	ID        string   `yaml:"id"`
	Kind      string   `yaml:"kind"`
	Name      string   `yaml:"name"`
	Template  string   `yaml:"template"`
	Variables []string `yaml:"variables"`
}

type Feature struct {
	ID        string    `yaml:"id"`
	Name      string    `yaml:"name"`
	Kinds     []string  `yaml:"kinds"`
	Stacks    []string  `yaml:"stacks"`
	Variants  []Variant `yaml:"variants"`
	Requires  []string  `yaml:"requires"`
	Conflicts []string  `yaml:"conflicts"`
}

type Variant struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Default bool   `yaml:"default"`
}

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

func LoadFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}
	return Parse(data)
}
