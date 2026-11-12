package patch

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func decodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	return result
}

func TestPatchJSON(t *testing.T) {
	t.Parallel()

	base := []byte(`{
  "name": "app",
  "scripts": {"test": "vitest"},
  "keywords": ["a"]
}`)

	t.Run("set nested value", func(t *testing.T) {
		t.Parallel()

		got, err := patchJSON(base, "set", "scripts.build", "vite build")
		if err != nil {
			t.Fatalf("patch: %v", err)
		}

		scripts := mustMap(t, decodeJSON(t, got)["scripts"], "scripts")
		if scripts["build"] != "vite build" {
			t.Fatalf("build = %v", scripts["build"])
		}
	})

	t.Run("merge object", func(t *testing.T) {
		t.Parallel()

		got, err := patchJSON(base, "merge", "scripts", map[string]any{"lint": "eslint ."})
		if err != nil {
			t.Fatalf("patch: %v", err)
		}

		scripts := mustMap(t, decodeJSON(t, got)["scripts"], "scripts")
		if scripts["lint"] != "eslint ." || scripts["test"] != "vitest" {
			t.Fatalf("scripts = %v", scripts)
		}
	})

	t.Run("append array", func(t *testing.T) {
		t.Parallel()

		got, err := patchJSON(base, "append", "keywords", "b")
		if err != nil {
			t.Fatalf("patch: %v", err)
		}

		keywords := mustSlice(t, decodeJSON(t, got)["keywords"], "keywords")
		if len(keywords) != 2 || keywords[1] != "b" {
			t.Fatalf("keywords = %v", keywords)
		}
	})

	t.Run("delete key", func(t *testing.T) {
		t.Parallel()

		got, err := patchJSON(base, "delete", "scripts.test", nil)
		if err != nil {
			t.Fatalf("patch: %v", err)
		}

		scripts := mustMap(t, decodeJSON(t, got)["scripts"], "scripts")
		if _, ok := scripts["test"]; ok {
			t.Fatalf("scripts still contains test: %v", scripts)
		}
	})

	t.Run("rejects unknown operation", func(t *testing.T) {
		t.Parallel()

		_, err := patchJSON(base, "explode", "scripts", nil)
		if err == nil || !strings.Contains(err.Error(), "operation") {
			t.Fatalf("error = %v, want operation error", err)
		}
	})
}

func TestPatchYAML(t *testing.T) {
	t.Parallel()

	base := []byte(`# development compose
services:
  api:
    image: app
`)

	t.Run("creates nested mapping and preserves comments", func(t *testing.T) {
		t.Parallel()

		got, err := patchYAML(base, "set", "services.redis", map[string]any{
			"image": "redis:7-alpine",
		})
		if err != nil {
			t.Fatalf("patch: %v", err)
		}

		text := string(got)
		if !strings.Contains(text, "# development compose") {
			t.Fatalf("comment was lost:\n%s", text)
		}

		var doc map[string]any
		if err := yaml.Unmarshal(got, &doc); err != nil {
			t.Fatalf("reparse: %v", err)
		}

		services := mustMap(t, doc["services"], "services")

		redis := mustMap(t, services["redis"], "redis")
		if redis["image"] != "redis:7-alpine" {
			t.Fatalf("redis = %v", redis)
		}
	})

	t.Run("deletes key", func(t *testing.T) {
		t.Parallel()

		got, err := patchYAML(base, "delete", "services.api", nil)
		if err != nil {
			t.Fatalf("patch: %v", err)
		}

		var doc map[string]any
		if err := yaml.Unmarshal(got, &doc); err != nil {
			t.Fatalf("reparse: %v", err)
		}

		services := mustMap(t, doc["services"], "services")
		if _, ok := services["api"]; ok {
			t.Fatalf("api was not deleted: %v", services)
		}
	})
}

func TestPatchTOML(t *testing.T) {
	t.Parallel()

	base := []byte("[package]\nname = \"app\"\nversion = \"0.1.0\"\n")

	got, err := patchTOML(base, "set", "dependencies.redis", map[string]any{
		"version": "0.27",
	})
	if err != nil {
		t.Fatalf("patch: %v", err)
	}

	var doc map[string]any
	if err := toml.Unmarshal(got, &doc); err != nil {
		t.Fatalf("reparse: %v", err)
	}

	dependencies := mustMap(t, doc["dependencies"], "dependencies")

	redis := mustMap(t, dependencies["redis"], "redis")
	if redis["version"] != "0.27" {
		t.Fatalf("redis = %v", redis)
	}
}
