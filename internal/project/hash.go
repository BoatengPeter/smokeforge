package project

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// HashBytes returns the SHA-256 of data with a "sha256:" prefix.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)

	return "sha256:" + hex.EncodeToString(sum[:])
}

// HashDir hashes a directory tree deterministically: sorted relative paths and
// file contents, including nested directories.
func HashDir(dir string) (string, error) {
	paths := make([]string, 0, 16)

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		if !entry.Type().IsRegular() {
			return fmt.Errorf("hash directory %q: unsupported entry %q", dir, path)
		}

		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("hash directory %q: %w", dir, err)
		}

		paths = append(paths, filepath.ToSlash(relative))

		return nil
	})
	if err != nil {
		return "", err
	}

	slices.Sort(paths)

	digest := sha256.New()
	for _, relative := range paths {
		if err := hashFile(digest, filepath.Join(dir, filepath.FromSlash(relative)), relative); err != nil {
			return "", err
		}
	}

	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), nil
}

func hashFile(digest hash.Hash, path, relative string) error {
	//nolint:gosec // path is derived from a directory walk rooted at a validated registry directory.
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("hash file %q: %w", path, err)
	}

	_, writeErr := io.WriteString(digest, relative+"\x00")
	if writeErr == nil {
		_, writeErr = io.Copy(digest, file)
	}

	if writeErr == nil {
		_, writeErr = io.WriteString(digest, "\x00")
	}

	closeErr := file.Close()

	if writeErr != nil {
		return fmt.Errorf("hash file %q: %w", path, writeErr)
	}

	if closeErr != nil {
		return fmt.Errorf("close file %q: %w", path, closeErr)
	}

	return nil
}

// HashFile returns the SHA-256 of one file with a "sha256:" prefix.
func HashFile(path string) (string, error) {
	//nolint:gosec // path is resolved from a validated generated project tree.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("hash file %q: %w", path, err)
	}

	if strings.TrimSpace(path) == "" {
		return "", errors.New("hash file: empty path")
	}

	return HashBytes(data), nil
}
