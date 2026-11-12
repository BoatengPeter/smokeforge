package render

import (
	"strings"
	"testing"
)

func TestSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "simple", input: "acme", want: "acme"},
		{name: "spaces", input: "My New App", want: "my-new-app"},
		{name: "punctuation", input: "  --Weird__Name!! ", want: "weird-name"},
		{name: "mixed case", input: "SaaS Starter", want: "saas-starter"},
		{name: "empty", input: "", want: "project"},
		{name: "symbols only", input: "!!!", want: "project"},
		{name: "numbers", input: "app 2", want: "app-2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Slug(tt.input)
			if got != tt.want {
				t.Fatalf("Slug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderBytes(t *testing.T) {
	t.Parallel()

	renderer := New("My App", map[string]string{
		"module_path": "example.com/myapp",
	})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "variables and builtins",
			input: "{{.project_name}} {{.project_slug}} {{.module_path}}",
			want:  "My App my-app example.com/myapp",
		},
		{
			name:  "snake function",
			input: `{{snake "HTTPServer"}}`,
			want:  "http_server",
		},
		{
			name:  "kebab function",
			input: `{{kebab "MyCoolApp"}}`,
			want:  "my-cool-app",
		},
		{
			name:  "camel function",
			input: `{{camel "my cool app"}}`,
			want:  "myCoolApp",
		},
		{
			name:  "pascal function",
			input: `{{pascal "my-cool-app"}}`,
			want:  "MyCoolApp",
		},
		{
			name:  "lower and upper",
			input: `{{upper "go"}}-{{lower "GO"}}`,
			want:  "GO-go",
		},
		{
			name:  "quote",
			input: `{{quote .module_path}}`,
			want:  `"example.com/myapp"`,
		},
		{
			name:  "default with value",
			input: `{{default "fallback" .module_path}}`,
			want:  "example.com/myapp",
		},
		{
			name:    "missing variable",
			input:   `{{.missing}}`,
			wantErr: "missing",
		},
		{
			name:    "invalid syntax",
			input:   `{{.project_name`,
			wantErr: "parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := renderer.RenderBytes([]byte(tt.input))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("render: %v", err)
			}

			if string(got) != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderPath(t *testing.T) {
	t.Parallel()

	renderer := New("My App", map[string]string{"package_name": "internal"})

	got, err := renderer.RenderPath("cmd/{{.project_slug}}/{{snake .package_name}}_test.go")
	if err != nil {
		t.Fatalf("render path: %v", err)
	}

	want := "cmd/my-app/internal_test.go"
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestNewCopiesVariables(t *testing.T) {
	t.Parallel()

	variables := map[string]string{"module_path": "example.com/original"}
	renderer := New("App", variables)
	variables["module_path"] = "example.com/changed"

	got, err := renderer.RenderBytes([]byte(`{{.module_path}}`))
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if string(got) != "example.com/original" {
		t.Fatalf("got %q, want %q", got, "example.com/original")
	}
}
