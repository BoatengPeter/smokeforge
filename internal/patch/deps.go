package patch

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/mod/modfile"
)

func addDependency(
	data []byte,
	file string,
	module string,
	version string,
	features []string,
) ([]byte, error) {
	switch strings.ToLower(filepath.Base(file)) {
	case "go.mod":
		return addGoDependency(data, module, version)
	case "package.json":
		return addNodeDependency(data, module, version)
	case "pyproject.toml":
		return addPythonDependency(data, module, version)
	case "cargo.toml":
		return addCargoDependency(data, module, version, features)
	default:
		return nil, fmt.Errorf("unsupported dependency file %q", file)
	}
}

func addGoDependency(data []byte, module, version string) ([]byte, error) {
	parsed, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}

	for _, requirement := range parsed.Require {
		if requirement.Mod.Path != module {
			continue
		}

		if requirement.Mod.Version == version {
			return data, nil
		}

		return nil, fmt.Errorf(
			"go.mod already requires %q at %q",
			module,
			requirement.Mod.Version,
		)
	}

	if err := parsed.AddRequire(module, version); err != nil {
		return nil, fmt.Errorf("add go.mod requirement: %w", err)
	}

	out, err := parsed.Format()
	if err != nil {
		return nil, fmt.Errorf("format go.mod: %w", err)
	}

	return out, nil
}

func addNodeDependency(data []byte, module, version string) ([]byte, error) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	dependencies, err := stringMap(document, "dependencies")
	if err != nil {
		return nil, err
	}

	if existing, ok := dependencies[module]; ok {
		if existing == version {
			return data, nil
		}

		return nil, fmt.Errorf("package.json already requires %q at %q", module, existing)
	}

	dependencies[module] = version

	out, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal package.json: %w", err)
	}

	return append(out, '\n'), nil
}

func stringMap(document map[string]any, key string) (map[string]any, error) {
	existing, ok := document[key]
	if !ok {
		created := make(map[string]any)
		document[key] = created

		return created, nil
	}

	mapping, ok := existing.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("package.json %q is not an object", key)
	}

	return mapping, nil
}

func addPythonDependency(data []byte, module, version string) ([]byte, error) {
	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse pyproject.toml: %w", err)
	}

	project, ok := document["project"].(map[string]any)
	if !ok {
		return nil, errors.New("pyproject.toml is missing the project table")
	}

	entry := module + "==" + version

	dependencies := make([]any, 0, 4)

	if existing, ok := project["dependencies"]; ok {
		list, ok := existing.([]any)
		if !ok {
			return nil, errors.New("pyproject.toml project.dependencies is not an array")
		}

		dependencies = list
	}

	for _, dependency := range dependencies {
		text, _ := dependency.(string)
		if text == entry || strings.HasPrefix(text, module+"==") {
			return data, nil
		}
	}

	project["dependencies"] = append(dependencies, entry)

	out, err := toml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("marshal pyproject.toml: %w", err)
	}

	return out, nil
}

func addCargoDependency(data []byte, module, version string, features []string) ([]byte, error) {
	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse Cargo.toml: %w", err)
	}

	dependencies, err := tomlDependencyMap(document)
	if err != nil {
		return nil, err
	}

	if existing, ok := dependencies[module]; ok {
		return nil, fmt.Errorf("cargo.toml already requires %q (%v)", module, existing)
	}

	if len(features) == 0 {
		dependencies[module] = version
	} else {
		featureValues := make([]any, 0, len(features))
		for _, feature := range features {
			featureValues = append(featureValues, feature)
		}

		dependencies[module] = map[string]any{
			"version":  version,
			"features": featureValues,
		}
	}

	out, err := toml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("marshal Cargo.toml: %w", err)
	}

	return out, nil
}

func tomlDependencyMap(document map[string]any) (map[string]any, error) {
	existing, ok := document["dependencies"]
	if !ok {
		created := make(map[string]any)
		document["dependencies"] = created

		return created, nil
	}

	mapping, ok := existing.(map[string]any)
	if !ok {
		return nil, errors.New("cargo.toml dependencies is not a table")
	}

	return mapping, nil
}
