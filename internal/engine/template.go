package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/render"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

type baseRenderer struct {
	root     *os.Root
	filesDir string
	renderer *render.Renderer
}

func renderBase(
	ctx context.Context,
	root *os.Root,
	plan *resolve.Plan,
	renderer *render.Renderer,
) ([]string, error) {
	filesDir := filepath.Join(plan.Target.TemplateDir, "files")

	info, err := os.Stat(filesDir)
	if err != nil {
		return nil, fmt.Errorf("read template files: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("template files path %q is not a directory", filesDir)
	}

	base := baseRenderer{root: root, filesDir: filesDir, renderer: renderer}
	files := make([]string, 0, 16)

	err = filepath.WalkDir(filesDir, func(source string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		destination, err := base.renderFile(source, entry)
		if err != nil {
			return err
		}

		files = append(files, destination)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("render base template: %w", err)
	}

	return files, nil
}

func (b baseRenderer) renderFile(source string, entry fs.DirEntry) (string, error) {
	if entry.Type()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("template file %q is a symlink", source)
	}

	if !entry.Type().IsRegular() {
		return "", fmt.Errorf("template entry %q is not a regular file", source)
	}

	relative, err := filepath.Rel(b.filesDir, source)
	if err != nil {
		return "", fmt.Errorf("resolve template path: %w", err)
	}

	isTemplate := strings.HasSuffix(relative, ".tmpl")
	destination := strings.TrimSuffix(filepath.ToSlash(relative), ".tmpl")

	destination, err = b.renderer.RenderPath(destination)
	if err != nil {
		return "", fmt.Errorf("render template path %q: %w", relative, err)
	}

	destination, err = safeRelativePath(destination)
	if err != nil {
		return "", err
	}

	//nolint:gosec // source is inside the validated template directory.
	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read template file %q: %w", source, err)
	}

	if isTemplate {
		data, err = b.renderer.RenderBytes(data)
		if err != nil {
			return "", fmt.Errorf("render template file %q: %w", relative, err)
		}
	}

	mode := os.FileMode(0o644)
	if info, infoErr := entry.Info(); infoErr == nil && info.Mode()&0o111 != 0 {
		mode = 0o755
	}

	if err := writeRootFile(b.root, destination, data, mode); err != nil {
		return "", err
	}

	return destination, nil
}

func writeRootFile(root *os.Root, destination string, data []byte, mode os.FileMode) error {
	if dir := path.Dir(destination); dir != "." {
		if err := root.MkdirAll(filepath.FromSlash(dir), 0o755); err != nil {
			return fmt.Errorf("create parent directory for %q: %w", destination, err)
		}
	}

	file, err := root.OpenFile(
		filepath.FromSlash(destination),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		mode,
	)
	if err != nil {
		return fmt.Errorf("create file %q: %w", destination, err)
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()

		return fmt.Errorf("write file %q: %w", destination, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file %q: %w", destination, err)
	}

	return nil
}

func safeRelativePath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("path must not be empty")
	}

	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("path %q must use forward slashes", value)
	}

	cleaned := path.Clean(value)
	if cleaned != value || path.IsAbs(value) || !filepath.IsLocal(filepath.FromSlash(value)) {
		return "", fmt.Errorf("path %q must be a safe relative path", value)
	}

	return cleaned, nil
}

func listFiles(root string) ([]string, error) {
	files := make([]string, 0, 32)

	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}

			return nil
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		relative, err := filepath.Rel(root, current)
		if err != nil {
			return fmt.Errorf("list generated files: %w", err)
		}

		files = append(files, filepath.ToSlash(relative))

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list generated files: %w", err)
	}

	sort.Strings(files)

	return files, nil
}
