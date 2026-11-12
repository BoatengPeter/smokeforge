package patch

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/mod/modfile"
)

func TestAddGoDependency(t *testing.T) {
	t.Parallel()

	base := []byte("module example.com/app\n\ngo 1.26\n")

	got, err := addDependency(base, "go.mod", "github.com/redis/go-redis/v9", "v9.7.0", nil)
	if err != nil {
		t.Fatalf("add dependency: %v", err)
	}

	parsed, err := modfile.Parse("go.mod", got, nil)
	if err != nil {
		t.Fatalf("parse go.mod: %v", err)
	}

	if len(parsed.Require) != 1 || parsed.Require[0].Mod.Path != "github.com/redis/go-redis/v9" {
		t.Fatalf("require = %+v", parsed.Require)
	}
}

func TestAddNodeDependency(t *testing.T) {
	t.Parallel()

	base := []byte("{\n  \"name\": \"app\"\n}\n")

	got, err := addDependency(base, "package.json", "ioredis", "^5.4.1", nil)
	if err != nil {
		t.Fatalf("add dependency: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatal(err)
	}

	dependencies := mustMap(t, doc["dependencies"], "dependencies")
	if dependencies["ioredis"] != "^5.4.1" {
		t.Fatalf("dependencies = %v", dependencies)
	}
}

func TestAddPythonDependency(t *testing.T) {
	t.Parallel()

	base := []byte(`[project]
name = "app"
dependencies = ["fastapi"]
`)

	got, err := addDependency(base, "pyproject.toml", "redis", "5.0.0", nil)
	if err != nil {
		t.Fatalf("add dependency: %v", err)
	}

	var doc map[string]any
	if err := toml.Unmarshal(got, &doc); err != nil {
		t.Fatal(err)
	}

	project := mustMap(t, doc["project"], "project")

	deps := mustSlice(t, project["dependencies"], "dependencies")
	if len(deps) != 2 || deps[1] != "redis==5.0.0" {
		t.Fatalf("dependencies = %v", deps)
	}
}

func TestAddCargoDependency(t *testing.T) {
	t.Parallel()

	base := []byte("[package]\nname = \"app\"\nversion = \"0.1.0\"\n")

	got, err := addDependency(base, "Cargo.toml", "redis", "0.27", []string{"tokio-comp"})
	if err != nil {
		t.Fatalf("add dependency: %v", err)
	}

	var doc map[string]any
	if err := toml.Unmarshal(got, &doc); err != nil {
		t.Fatal(err)
	}

	dependencies := mustMap(t, doc["dependencies"], "dependencies")

	redis := mustMap(t, dependencies["redis"], "redis")
	if redis["version"] != "0.27" {
		t.Fatalf("redis = %v", redis)
	}

	features := mustSlice(t, redis["features"], "features")
	if len(features) != 1 || features[0] != "tokio-comp" {
		t.Fatalf("features = %v", features)
	}
}

func TestAddDependencyUnsupported(t *testing.T) {
	t.Parallel()

	_, err := addDependency([]byte("x"), "requirements.txt", "redis", "5", nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error = %v, want unsupported error", err)
	}
}
