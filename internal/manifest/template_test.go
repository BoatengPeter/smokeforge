package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validTemplate = `id: test/plain
variables:
  - name: module_path
    prompt: Go module path
    default: example.com/app
  - name: greeting
    prompt: Greeting
    default: hello
`

func writeTemplateFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "template.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoadTemplateFile(t *testing.T) {
	t.Parallel()

	tmpl, err := LoadTemplateFile(writeTemplateFile(t, validTemplate))
	if err != nil {
		t.Fatalf("load template: %v", err)
	}

	if tmpl.ID != "test/plain" {
		t.Fatalf("id = %q, want test/plain", tmpl.ID)
	}

	if len(tmpl.Variables) != 2 {
		t.Fatalf("variables = %d, want 2", len(tmpl.Variables))
	}
}

func TestLoadTemplateFileRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	path := writeTemplateFile(t, "id: test/plain\nunknown: true\n")
	if _, err := LoadTemplateFile(path); err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestTemplate_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Template)
		want   string
	}{
		{
			name:   "empty id",
			mutate: func(tmpl *Template) { tmpl.ID = "" },
			want:   "id",
		},
		{
			name:   "empty variable name",
			mutate: func(tmpl *Template) { tmpl.Variables[0].Name = "" },
			want:   "name",
		},
		{
			name:   "invalid variable name",
			mutate: func(tmpl *Template) { tmpl.Variables[0].Name = "ModulePath" },
			want:   "name",
		},
		{
			name: "duplicate variable",
			mutate: func(tmpl *Template) {
				tmpl.Variables = append(tmpl.Variables, tmpl.Variables[0])
			},
			want: "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl := &Template{
				ID: "test/plain",
				Variables: []TemplateVariable{
					{Name: "module_path", Prompt: "Module", Default: "example.com/app"},
				},
			}
			tt.mutate(tmpl)

			err := tmpl.Validate()
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestTemplate_ValidateChoices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		choices []string
		def     string
		want    string
	}{
		{
			name:    "valid",
			choices: []string{"pnpm", "npm", "bun"},
			def:     "pnpm",
		},
		{
			name:    "default not a choice",
			choices: []string{"pnpm", "npm"},
			def:     "yarn",
			want:    "choices",
		},
		{
			name:    "duplicate choice",
			choices: []string{"pnpm", "pnpm"},
			def:     "pnpm",
			want:    "duplicate choice",
		},
		{
			name:    "empty choice",
			choices: []string{"pnpm", ""},
			def:     "pnpm",
			want:    "empty choice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl := &Template{
				ID: "next",
				Variables: []TemplateVariable{
					{Name: "package_manager", Prompt: "Package manager", Default: tt.def, Choices: tt.choices},
				},
			}

			assertValidateError(t, tmpl, tt.want)
		})
	}
}

func TestTemplate_ValidatePattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		def     string
		want    string
	}{
		{name: "valid", pattern: `[a-z][a-z0-9_]*`, def: "example_app"},
		{name: "invalid regex", pattern: `([a-z`, def: "example_app", want: "invalid pattern"},
		{name: "default does not match", pattern: `[a-z]+`, def: "Example App", want: "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpl := &Template{
				ID: "desktop/egui",
				Variables: []TemplateVariable{
					{Name: "crate_name", Prompt: "Crate", Default: tt.def, Pattern: tt.pattern},
				},
			}

			assertValidateError(t, tmpl, tt.want)
		})
	}
}

func TestTemplateVariable_CheckValue(t *testing.T) {
	t.Parallel()

	crate := TemplateVariable{Name: "crate_name", Pattern: `[a-z][a-z0-9_]*`}
	manager := TemplateVariable{Name: "package_manager", Choices: []string{"pnpm", "npm"}}
	free := TemplateVariable{Name: "site_description"}

	tests := []struct {
		name     string
		variable TemplateVariable
		value    string
		wantErr  bool
	}{
		{name: "pattern match", variable: crate, value: "my_app"},
		{name: "pattern must match the whole value", variable: crate, value: "my_app\"); evil()", wantErr: true},
		{name: "pattern partial prefix", variable: crate, value: "abc-1", wantErr: true},
		{name: "shell substitution", variable: crate, value: "$(id)", wantErr: true},
		{name: "choice accepted", variable: manager, value: "npm"},
		{name: "choice rejected", variable: manager, value: "yarn", wantErr: true},
		{name: "no constraints", variable: free, value: "Anything, really."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.variable.CheckValue(tt.value)
			if tt.wantErr && err == nil {
				t.Fatalf("CheckValue(%q) succeeded, want error", tt.value)
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("CheckValue(%q): %v", tt.value, err)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.variable.Name) {
				t.Fatalf("error %q should name variable %q", err, tt.variable.Name)
			}
		})
	}
}

// assertValidateError checks that tmpl validates when want is empty, and
// otherwise fails with an error containing want.
func assertValidateError(t *testing.T, tmpl *Template, want string) {
	t.Helper()

	err := tmpl.Validate()
	if want == "" {
		if err != nil {
			t.Fatalf("validate: %v", err)
		}

		return
	}

	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}
