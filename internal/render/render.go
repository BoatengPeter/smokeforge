// Package render renders strict text templates for project files and paths.
package render

import (
	"bytes"
	"fmt"
	"maps"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

// Renderer renders templates with a fixed variable set. Missing variables are
// errors, so generated projects never contain empty placeholders.
type Renderer struct {
	projectName string
	projectSlug string
	variables   map[string]string
}

// New creates a renderer with built-in project_name and project_slug variables.
// The provided map is copied so later caller mutations cannot affect rendering.
func New(projectName string, variables map[string]string) *Renderer {
	return &Renderer{
		projectName: projectName,
		projectSlug: Slug(projectName),
		variables:   maps.Clone(variables),
	}
}

// RenderPath renders a relative output path.
func (r *Renderer) RenderPath(value string) (string, error) {
	return r.render("path", value)
}

// RenderBytes renders file content.
func (r *Renderer) RenderBytes(content []byte) ([]byte, error) {
	rendered, err := r.render("file", string(content))
	if err != nil {
		return nil, err
	}

	return []byte(rendered), nil
}

func (r *Renderer) render(name, content string) (string, error) {
	tmpl, err := template.New(name).
		Funcs(r.funcMap()).
		Option("missingkey=error").
		Parse(content)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, r.data()); err != nil {
		return "", fmt.Errorf("render template %q: %w", name, err)
	}

	return buf.String(), nil
}

func (r *Renderer) data() map[string]string {
	data := maps.Clone(r.variables)
	if data == nil {
		data = make(map[string]string)
	}

	data["project_name"] = r.projectName
	data["project_slug"] = r.projectSlug

	return data
}

func (r *Renderer) funcMap() template.FuncMap {
	return template.FuncMap{
		"lower":   strings.ToLower,
		"upper":   strings.ToUpper,
		"snake":   snake,
		"kebab":   kebab,
		"camel":   camel,
		"pascal":  pascal,
		"quote":   strconv.Quote,
		"replace": strings.ReplaceAll,
		"packagePath": func(value string) string {
			return strings.ReplaceAll(value, ".", "/")
		},
		"default": func(fallback, value string) string {
			if value == "" {
				return fallback
			}

			return value
		},
	}
}

// Slug converts a display name into a lowercase, hyphen-separated identifier.
// It returns "project" when the input has no usable characters.
func Slug(value string) string {
	var slug strings.Builder

	lastWasSeparator := true

	for _, r := range strings.ToLower(value) {
		isASCIILetter := r >= 'a' && r <= 'z'

		isDigit := r >= '0' && r <= '9'
		if isASCIILetter || isDigit {
			slug.WriteRune(r)

			lastWasSeparator = false

			continue
		}

		if !lastWasSeparator {
			slug.WriteByte('-')

			lastWasSeparator = true
		}
	}

	result := strings.TrimSuffix(slug.String(), "-")
	if result == "" {
		return "project"
	}

	return result
}

func snake(value string) string {
	return strings.Join(splitWords(value), "_")
}

func kebab(value string) string {
	return strings.Join(splitWords(value), "-")
}

func camel(value string) string {
	words := splitWords(value)
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString(words[0])

	for _, word := range words[1:] {
		result.WriteString(capitalize(word))
	}

	return result.String()
}

func pascal(value string) string {
	words := splitWords(value)

	var result strings.Builder
	for _, word := range words {
		result.WriteString(capitalize(word))
	}

	return result.String()
}

func capitalize(value string) string {
	if value == "" {
		return value
	}

	return strings.ToUpper(value[:1]) + value[1:]
}

func splitWords(value string) []string {
	runes := []rune(value)
	words := make([]string, 0, 4)

	var current strings.Builder

	flush := func() {
		if current.Len() == 0 {
			return
		}

		words = append(words, strings.ToLower(current.String()))
		current.Reset()
	}

	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()

			continue
		}

		if unicode.IsUpper(r) && current.Len() > 0 {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

			prevIsLowerOrDigit := unicode.IsLower(prev) || unicode.IsDigit(prev)
			if prevIsLowerOrDigit || (unicode.IsUpper(prev) && nextIsLower) {
				flush()
			}
		}

		current.WriteRune(r)
	}

	flush()

	return words
}
