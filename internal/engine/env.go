package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/patch"
	"github.com/BoatengPeter/smokeforge/internal/render"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

type overlayApplication struct {
	FeatureID   string
	VariantID   string
	Path        string
	Dir         string
	Files       []string
	Markers     []string
	RegistryTag string
}

func applyOverlays(
	ctx context.Context,
	staging string,
	plan *resolve.Plan,
	renderer *render.Renderer,
) ([]overlayApplication, []patch.Env, error) {
	applications := make([]overlayApplication, 0, len(plan.Target.Overlays))
	envEntries := make([]patch.Env, 0, len(plan.Target.Overlays))

	for _, overlayPlan := range plan.Target.Overlays {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("apply overlays: %w", err)
		}

		applied, err := patch.Apply(ctx, overlayPlan.Overlay, patch.ApplyOptions{
			Root:       staging,
			OverlayDir: overlayPlan.Dir,
			Renderer:   renderer,
		})
		if err != nil {
			return nil, nil, err
		}

		applications = append(applications, overlayApplication{
			FeatureID: overlayPlan.FeatureID,
			VariantID: overlayPlan.VariantID,
			Path:      overlayPlan.Path,
			Dir:       overlayPlan.Dir,
			Files:     applied.Files,
			Markers:   applied.Markers,
		})
		envEntries = append(envEntries, applied.Env...)
	}

	return applications, envEntries, nil
}

func writeEnvFiles(root *os.Root, entries []patch.Env) error {
	example, err := readRootFile(root, ".env.example")
	if err != nil {
		return err
	}

	lines := splitConfigLines(example)
	index := configLineIndex(lines)

	for _, entry := range entries {
		if _, exists := index[entry.Key]; exists {
			continue
		}

		lines = append(lines, entry.Key+"="+entry.Example)
		index[entry.Key] = len(lines) - 1
	}

	exampleContent := renderLines(lines)
	if err := writeRootFile(root, ".env.example", []byte(exampleContent), 0o644); err != nil {
		return err
	}

	envLines := splitConfigLines(exampleContent)
	envIndex := configLineIndex(envLines)

	for _, entry := range entries {
		value := entry.Example
		if entry.Secret {
			value, err = randomSecret()
			if err != nil {
				return err
			}
		}

		line := entry.Key + "=" + value
		if position, exists := envIndex[entry.Key]; exists {
			envLines[position] = line

			continue
		}

		envLines = append(envLines, line)
		envIndex[entry.Key] = len(envLines) - 1
	}

	if err := writeRootFile(root, ".env", []byte(renderLines(envLines)), 0o600); err != nil {
		return err
	}

	return ensureGitignore(root)
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
