package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type archiveEntry struct {
	name     string
	body     string
	typeflag byte
	linkname string
}

func buildArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()

	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o644,
			Typeflag: entry.typeflag,
			Linkname: entry.linkname,
			Size:     int64(len(entry.body)),
		}
		if entry.typeflag == tar.TypeDir {
			header.Mode = 0o755
			header.Size = 0
		}

		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("write header %q: %v", entry.name, err)
		}

		if entry.body != "" {
			if _, err := tw.Write([]byte(entry.body)); err != nil {
				t.Fatalf("write body %q: %v", entry.name, err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}

	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	return buf.Bytes()
}

func TestExtract(t *testing.T) {
	t.Parallel()

	t.Run("writes regular files", func(t *testing.T) {
		t.Parallel()

		dst := t.TempDir()
		archive := buildArchive(t, []archiveEntry{
			{name: "manifest.yaml", body: "schemaVersion: 1\n", typeflag: tar.TypeReg},
			{name: "templates/go/README.md", body: "hello\n", typeflag: tar.TypeReg},
		})

		if err := Extract(t.Context(), bytes.NewReader(archive), dst); err != nil {
			t.Fatalf("extract: %v", err)
		}

		//nolint:gosec // test reads a known file under t.TempDir().
		got, err := os.ReadFile(filepath.Join(dst, "templates", "go", "README.md"))
		if err != nil {
			t.Fatal(err)
		}

		if string(got) != "hello\n" {
			t.Fatalf("content = %q, want %q", got, "hello\n")
		}
	})

	t.Run("creates directories", func(t *testing.T) {
		t.Parallel()

		dst := t.TempDir()
		archive := buildArchive(t, []archiveEntry{
			{name: "templates/", typeflag: tar.TypeDir},
			{name: "templates/go/", typeflag: tar.TypeDir},
		})

		if err := Extract(t.Context(), bytes.NewReader(archive), dst); err != nil {
			t.Fatalf("extract: %v", err)
		}

		info, err := os.Stat(filepath.Join(dst, "templates", "go"))
		if err != nil {
			t.Fatal(err)
		}

		if !info.IsDir() {
			t.Fatalf("%q is not a directory", info.Name())
		}
	})
}

func TestExtractRejectsUnsafeArchive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		entry archiveEntry
	}{
		{
			name:  "parent traversal",
			entry: archiveEntry{name: "../escape", body: "x", typeflag: tar.TypeReg},
		},
		{
			name:  "nested parent traversal",
			entry: archiveEntry{name: "a/../../escape", body: "x", typeflag: tar.TypeReg},
		},
		{
			name:  "absolute path",
			entry: archiveEntry{name: "/etc/passwd", body: "x", typeflag: tar.TypeReg},
		},
		{
			name:  "symlink",
			entry: archiveEntry{name: "link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"},
		},
		{
			name:  "hardlink",
			entry: archiveEntry{name: "link", typeflag: tar.TypeLink, linkname: "manifest.yaml"},
		},
		{
			name:  "unsupported type",
			entry: archiveEntry{name: "device", typeflag: tar.TypeChar},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dst := t.TempDir()
			archive := buildArchive(t, []archiveEntry{tt.entry})

			err := Extract(t.Context(), bytes.NewReader(archive), dst)
			if !errors.Is(err, ErrUnsafeArchive) {
				t.Fatalf("error = %v, want ErrUnsafeArchive", err)
			}
		})
	}
}

func TestExtractHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	archive := buildArchive(t, []archiveEntry{
		{name: "manifest.yaml", body: "x", typeflag: tar.TypeReg},
	})

	err := Extract(ctx, bytes.NewReader(archive), t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestValidateEntry(t *testing.T) {
	t.Parallel()

	t.Run("accepts regular file", func(t *testing.T) {
		t.Parallel()

		header := &tar.Header{Name: "manifest.yaml", Typeflag: tar.TypeReg, Size: 10}
		if _, err := validateEntry(header); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects oversized file", func(t *testing.T) {
		t.Parallel()

		header := &tar.Header{
			Name:     "huge.bin",
			Typeflag: tar.TypeReg,
			Size:     maxArchiveFileSize + 1,
		}
		if _, err := validateEntry(header); !errors.Is(err, ErrUnsafeArchive) {
			t.Fatalf("error = %v, want ErrUnsafeArchive", err)
		}
	})
}
