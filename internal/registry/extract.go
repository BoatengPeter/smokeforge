package registry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

const (
	maxArchiveEntries   = 10_000
	maxArchiveFileSize  = 100 << 20
	maxArchiveTotalSize = 512 << 20
)

// Extract safely unpacks a gzip-compressed registry archive into dst. Entries
// that escape dst, links, devices, and oversized files are rejected. The
// context is checked between entries so cancellation stops extraction promptly.
func Extract(ctx context.Context, archive io.Reader, dst string) error {
	gz, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("open registry archive: %w", err)
	}

	defer func() {
		_ = gz.Close()
	}()

	if err := os.MkdirAll(dst, 0o750); err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}

	root, err := os.OpenRoot(dst)
	if err != nil {
		return fmt.Errorf("open extraction root: %w", err)
	}

	defer func() {
		_ = root.Close()
	}()

	reader := tar.NewReader(gz)
	stats := archiveStats{}

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("extract registry: %w", err)
		}

		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("read registry archive: %w", err)
		}

		if err := stats.add(header); err != nil {
			return err
		}

		if err := extractEntry(root, reader, header); err != nil {
			return err
		}
	}
}

type archiveStats struct {
	entries int
	total   int64
}

func (s *archiveStats) add(header *tar.Header) error {
	s.entries++
	if s.entries > maxArchiveEntries {
		return fmt.Errorf("%w: more than %d entries", ErrUnsafeArchive, maxArchiveEntries)
	}

	if header.Typeflag != tar.TypeReg {
		return nil
	}

	s.total += header.Size
	if s.total > maxArchiveTotalSize {
		return fmt.Errorf(
			"%w: archive exceeds %d bytes",
			ErrUnsafeArchive,
			maxArchiveTotalSize,
		)
	}

	return nil
}

func extractEntry(root *os.Root, reader io.Reader, header *tar.Header) error {
	name, err := validateEntry(header)
	if err != nil {
		return err
	}

	if header.Typeflag == tar.TypeDir {
		if err := root.MkdirAll(name, 0o750); err != nil {
			return fmt.Errorf("create directory %q: %w", header.Name, err)
		}

		return nil
	}

	return writeEntry(root, name, reader, header.Size)
}

func validateEntry(header *tar.Header) (string, error) {
	name := path.Clean(header.Name)
	local := filepath.FromSlash(name)

	isRootEntry := name == "."
	isAbsent := path.IsAbs(name)

	isEscaping := !filepath.IsLocal(local)
	if isRootEntry || isAbsent || isEscaping {
		return "", fmt.Errorf("%w: path %q", ErrUnsafeArchive, header.Name)
	}

	switch header.Typeflag {
	case tar.TypeDir:
		return local, nil
	case tar.TypeReg:
	default:
		return "", fmt.Errorf(
			"%w: unsupported entry type %d for %q",
			ErrUnsafeArchive,
			header.Typeflag,
			header.Name,
		)
	}

	if header.Size < 0 || header.Size > maxArchiveFileSize {
		return "", fmt.Errorf(
			"%w: file %q is %d bytes",
			ErrUnsafeArchive,
			header.Name,
			header.Size,
		)
	}

	return local, nil
}

func writeEntry(root *os.Root, name string, reader io.Reader, size int64) error {
	if dir := path.Dir(filepath.ToSlash(name)); dir != "." {
		if err := root.MkdirAll(filepath.FromSlash(dir), 0o750); err != nil {
			return fmt.Errorf("create parent directory for %q: %w", name, err)
		}
	}

	file, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create file %q: %w", name, err)
	}

	if _, err := io.CopyN(file, reader, size); err != nil {
		_ = file.Close()

		return fmt.Errorf("write file %q: %w", name, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file %q: %w", name, err)
	}

	return nil
}
