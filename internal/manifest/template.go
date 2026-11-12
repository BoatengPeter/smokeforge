package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var variableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Template describes a base template's identity and the variables it accepts.
type Template struct {
	ID        string             `yaml:"id"`
	Variables []TemplateVariable `yaml:"variables"`
}

// TemplateVariable declares one template input, its prompt, and its default.
type TemplateVariable struct {
	Name    string   `yaml:"name"`
	Prompt  string   `yaml:"prompt"`
	Default string   `yaml:"default"`
	Choices []string `yaml:"choices"`
}

// LoadTemplateFile reads and parses a template metadata file.
func LoadTemplateFile(path string) (*Template, error) {
	//nolint:gosec // path is resolved from a validated registry template directory.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", path, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var tmpl Template
	if err := dec.Decode(&tmpl); err != nil {
		return nil, fmt.Errorf("parse template %q: %w", path, err)
	}

	var extra any

	err = dec.Decode(&extra)
	switch {
	case err == nil:
		return nil, fmt.Errorf("parse template %q: multiple YAML documents are not supported", path)
	case errors.Is(err, io.EOF):
		return &tmpl, nil
	default:
		return nil, fmt.Errorf("parse template %q: %w", path, err)
	}
}

// Validate checks the template ID, variable names, and uniqueness rules.
func (t *Template) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return errors.New("template id must not be empty")
	}

	if len(t.Variables) == 0 {
		return errors.New("template variables must not be empty")
	}

	seen := make(map[string]bool, len(t.Variables))
	for _, variable := range t.Variables {
		if !variableNamePattern.MatchString(variable.Name) {
			return fmt.Errorf(
				"template %q: invalid variable name %q",
				t.ID,
				variable.Name,
			)
		}

		if seen[variable.Name] {
			return fmt.Errorf("template %q: duplicate variable %q", t.ID, variable.Name)
		}

		seen[variable.Name] = true

		if err := validateChoices(t.ID, variable); err != nil {
			return err
		}
	}

	return nil
}

func validateChoices(templateID string, variable TemplateVariable) error {
	if len(variable.Choices) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(variable.Choices))
	for _, choice := range variable.Choices {
		if strings.TrimSpace(choice) == "" {
			return fmt.Errorf(
				"template %q: variable %q has an empty choice",
				templateID,
				variable.Name,
			)
		}

		if seen[choice] {
			return fmt.Errorf(
				"template %q: variable %q has duplicate choice %q",
				templateID,
				variable.Name,
				choice,
			)
		}

		seen[choice] = true
	}

	if variable.Default == "" || !seen[variable.Default] {
		return fmt.Errorf(
			"template %q: variable %q default %q must be one of its choices",
			templateID,
			variable.Name,
			variable.Default,
		)
	}

	return nil
}
