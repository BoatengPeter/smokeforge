package engine

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/patch"
)

type overlayApplication struct {
	FeatureID   string
	VariantID   string
	Path        string
	Dir         string
	Files       []string
	Markers     []string
	Env         []patch.Env
	RegistryTag string
}

func writeEnvFiles(root *os.Root, entries []patch.Env) error {
	example, err := readRootFile(root, ".env.example")
	if err != nil {
		return err
	}

	exampleLines := splitConfigLines(example)
	exampleLines = addExampleEntries(exampleLines, entries)

	exampleContent := renderLines(exampleLines)
	if err := writeRootFile(root, ".env.example", []byte(exampleContent), 0o644); err != nil {
		return err
	}

	env, err := readRootFile(root, ".env")
	if err != nil {
		return err
	}

	envLines := splitConfigLines(env)

	envFromExample := len(envLines) == 0
	if envFromExample {
		envLines = splitConfigLines(exampleContent)
	}

	envLines, err = addEnvValues(envLines, entries, envFromExample)
	if err != nil {
		return err
	}

	if err := writeRootFile(root, ".env", []byte(renderLines(envLines)), 0o600); err != nil {
		return err
	}

	return ensureGitignore(root)
}

func addExampleEntries(lines []string, entries []patch.Env) []string {
	index := configLineIndex(lines)

	for _, entry := range entries {
		if _, exists := index[entry.Key]; exists {
			continue
		}

		lines = append(lines, entry.Key+"="+entry.Example)
		index[entry.Key] = len(lines) - 1
	}

	return lines
}

func addEnvValues(lines []string, entries []patch.Env, fromExample bool) ([]string, error) {
	index := configLineIndex(lines)

	for _, entry := range entries {
		if position, exists := index[entry.Key]; exists {
			if !fromExample || !entry.Secret {
				continue
			}

			value, err := randomSecret()
			if err != nil {
				return nil, err
			}

			lines[position] = entry.Key + "=" + value

			continue
		}

		value := entry.Example
		if entry.Secret {
			var err error

			value, err = randomSecret()
			if err != nil {
				return nil, err
			}
		}

		lines = append(lines, entry.Key+"="+value)
		index[entry.Key] = len(lines) - 1
	}

	return lines, nil
}

func ensureGitignore(root *os.Root) error {
	content, err := readRootFile(root, ".gitignore")
	if err != nil {
		return err
	}

	lines := splitConfigLines(content)

	present := make(map[string]bool, len(lines))
	for _, line := range lines {
		present[strings.TrimSpace(line)] = true
	}

	for _, entry := range []string{".env", ".smokeforge/"} {
		if !present[entry] {
			lines = append(lines, entry)
			present[entry] = true
		}
	}

	return writeRootFile(root, ".gitignore", []byte(renderLines(lines)), 0o644)
}

func readRootFile(root *os.Root, name string) (string, error) {
	data, err := root.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("read %q: %w", name, err)
	}

	return string(data), nil
}

func splitConfigLines(content string) []string {
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return make([]string, 0, 4)
	}

	return strings.Split(content, "\n")
}

func renderLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n"
}

func configLineIndex(lines []string) map[string]int {
	index := make(map[string]int, len(lines))
	for position, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		key, _, found := strings.Cut(trimmed, "=")
		if !found {
			continue
		}

		index[strings.TrimSpace(key)] = position
	}

	return index
}

func randomSecret() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}

	return hex.EncodeToString(buffer), nil
}
