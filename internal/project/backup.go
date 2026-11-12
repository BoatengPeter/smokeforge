package project

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Backup snapshots files before a mutation so a failure can restore them.
type Backup struct {
	root  string
	dir   string
	files map[string]bool
}

// CreateBackup copies the listed project-relative files into a timestamped
// backup directory. Missing files are recorded so Restore removes them.
func CreateBackup(root string, files []string) (*Backup, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	backupsDir := filepath.Join(absolute, ".smokeforge", "backups")

	if err := os.MkdirAll(backupsDir, 0o750); err != nil {
		return nil, fmt.Errorf("create backups directory: %w", err)
	}

	dir, err := os.MkdirTemp(backupsDir, time.Now().UTC().Format("20060102T150405.000000000")+"-*")
	if err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	backup := &Backup{
		root:  absolute,
		dir:   dir,
		files: make(map[string]bool, len(files)),
	}

	for _, raw := range files {
		if err := backup.add(raw); err != nil {
			_ = backup.Discard()

			return nil, err
		}
	}

	return backup, nil
}

func (b *Backup) add(raw string) error {
	clean, err := safeTrackedPath(raw)
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	if _, seen := b.files[clean]; seen {
		return nil
	}

	source := filepath.Join(b.root, filepath.FromSlash(clean))

	info, statErr := os.Stat(source)
	if errors.Is(statErr, os.ErrNotExist) {
		b.files[clean] = false

		return nil
	}

	if statErr != nil {
		return fmt.Errorf("backup stat %q: %w", clean, statErr)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("backup %q: not a regular file", clean)
	}

	destination := filepath.Join(b.dir, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return fmt.Errorf("backup create directory for %q: %w", clean, err)
	}

	if err := copyFile(source, destination, info.Mode().Perm()); err != nil {
		return err
	}

	b.files[clean] = true

	return nil
}

// Restore copies backed-up files back and removes files that did not exist
// before the mutation. All restorable files are attempted before returning.
func (b *Backup) Restore() error {
	problems := make([]error, 0, len(b.files))

	for raw, existed := range b.files {
		clean, err := safeTrackedPath(raw)
		if err != nil {
			problems = append(problems, err)

			continue
		}

		target := filepath.Join(b.root, filepath.FromSlash(clean))

		if !existed {
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				problems = append(problems, fmt.Errorf("restore remove %q: %w", clean, err))
			}

			continue
		}

		source := filepath.Join(b.dir, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			problems = append(problems, fmt.Errorf("restore directory for %q: %w", clean, err))

			continue
		}

		info, statErr := os.Stat(source)
		if statErr != nil {
			problems = append(problems, fmt.Errorf("restore stat %q: %w", clean, statErr))

			continue
		}

		if err := copyFile(source, target, info.Mode().Perm()); err != nil {
			problems = append(problems, err)
		}
	}

	return errors.Join(problems...)
}

// Discard removes the backup directory.
func (b *Backup) Discard() error {
	if err := os.RemoveAll(b.dir); err != nil {
		return fmt.Errorf("discard backup: %w", err)
	}

	return nil
}

// Dir returns the backup directory path.
func (b *Backup) Dir() string {
	return b.dir
}

func copyFile(source, destination string, mode os.FileMode) error {
	//nolint:gosec // source is a validated file under the project or backup directory.
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open %q: %w", source, err)
	}

	defer func() {
		_ = input.Close()
	}()

	//nolint:gosec // destination is inside the project or its backup directory.
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create %q: %w", destination, err)
	}

	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()

		return fmt.Errorf("write %q: %w", destination, err)
	}

	if err := output.Close(); err != nil {
		return fmt.Errorf("close %q: %w", destination, err)
	}

	return nil
}
