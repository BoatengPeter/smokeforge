package patch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/render"
)

// RemoveOptions configures one overlay removal.
type RemoveOptions struct {
	Root       string
	OverlayDir string
	Renderer   *render.Renderer
}

// CheckRemovable rejects overlays whose operations cannot be reversed safely.
func CheckRemovable(overlay *Overlay) error {
	for index, op := range overlay.Ops {
		switch {
		case op.PatchJSON != nil:
			return nonReversible(overlay.ID, index, "patchJSON")
		case op.PatchYAML != nil:
			return nonReversible(overlay.ID, index, "patchYAML")
		case op.PatchTOML != nil:
			return nonReversible(overlay.ID, index, "patchTOML")
		case op.AddDependency != nil:
			return nonReversible(overlay.ID, index, "addDependency")
		}
	}

	return nil
}

func nonReversible(overlayID string, index int, operation string) error {
	return fmt.Errorf(
		"overlay %q: op %d uses non-reversible operation %q; remove it manually",
		overlayID,
		index,
		operation,
	)
}

// Remove reverts a reversible overlay in reverse operation order. Marker blocks
// are excised idempotently and overlay files are deleted. Environment keys are
// handled by the caller because they span shared env files.
func Remove(ctx context.Context, overlay *Overlay, opts RemoveOptions) error {
	if err := overlay.Validate(); err != nil {
		return err
	}

	if err := CheckRemovable(overlay); err != nil {
		return err
	}

	if opts.Renderer == nil {
		return errors.New("remove overlay: renderer is required")
	}

	root, err := os.OpenRoot(opts.Root)
	if err != nil {
		return fmt.Errorf("open project root: %w", err)
	}

	defer func() {
		_ = root.Close()
	}()

	for index := len(overlay.Ops) - 1; index >= 0; index-- {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("remove overlay %q: op %d: %w", overlay.ID, index, err)
		}

		if err := removeOp(root, overlay, opts, overlay.Ops[index]); err != nil {
			return fmt.Errorf("remove overlay %q: op %d: %w", overlay.ID, index, err)
		}
	}

	return nil
}

func removeOp(root *os.Root, overlay *Overlay, opts RemoveOptions, op Op) error {
	switch {
	case op.AddFiles != nil:
		return removeAddedFiles(root, opts.Renderer, *op.AddFiles)
	case op.InsertBefore != nil:
		return removeInsert(root, overlay, opts, op.InsertBefore)
	case op.InsertAfter != nil:
		return removeInsert(root, overlay, opts, op.InsertAfter)
	case op.Env != nil:
		return nil
	default:
		return nonReversible(overlay.ID, 0, "unknown")
	}
}

func removeAddedFiles(root *os.Root, renderer *render.Renderer, files AddFiles) error {
	for _, file := range files {
		dest, err := renderer.RenderPath(file.Dest)
		if err != nil {
			return fmt.Errorf("render overlay destination %q: %w", file.Dest, err)
		}

		dest, err = safeRelativePath(dest)
		if err != nil {
			return err
		}

		rootPath := filepath.FromSlash(dest)
		if err := root.Remove(rootPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete overlay file %q: %w", dest, err)
		}
	}

	return nil
}

func removeInsert(
	root *os.Root,
	overlay *Overlay,
	opts RemoveOptions,
	insert *Insert,
) error {
	target, err := opts.Renderer.RenderPath(insert.File)
	if err != nil {
		return fmt.Errorf("render insert file %q: %w", insert.File, err)
	}

	data, mode, err := readFile(root, target)
	if err != nil {
		return err
	}

	marker := insert.Marker
	if marker == "" {
		marker = overlay.ID
	}

	updated, err := removeBlock(string(data), marker)
	if err != nil {
		return err
	}

	if updated == string(data) {
		return nil
	}

	return writeFile(root, target, []byte(updated), mode)
}

func removeBlock(content, marker string) (string, error) {
	if strings.TrimSpace(marker) == "" {
		return "", errors.New("marker must not be empty")
	}

	beginToken := "smokeforge:begin " + marker
	endToken := "smokeforge:end " + marker
	lines := strings.Split(content, "\n")

	begin := -1
	end := -1

	for index, line := range lines {
		if strings.Contains(line, beginToken) {
			if begin >= 0 {
				return "", fmt.Errorf("marker %q appears more than once", marker)
			}

			begin = index
		}

		if strings.Contains(line, endToken) {
			if end >= 0 {
				return "", fmt.Errorf("marker %q end appears more than once", marker)
			}

			end = index
		}
	}

	if begin == -1 && end == -1 {
		return content, nil
	}

	if begin == -1 || end == -1 || end < begin {
		return "", fmt.Errorf("marker %q is malformed", marker)
	}

	result := make([]string, 0, len(lines)-(end-begin+1))
	result = append(result, lines[:begin]...)
	result = append(result, lines[end+1:]...)

	return strings.Join(result, "\n"), nil
}
