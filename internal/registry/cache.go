package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BoatengPeter/smokeforge/internal/version"
)

// DefaultCacheDir returns the per-user cache directory used for registries.
func DefaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory: %w", err)
	}

	return filepath.Join(base, "smokeforge", "registry"), nil
}

func entryDir(cacheDir, source, tag string) string {
	sum := sha256.Sum256([]byte(source + "\x00" + tag))

	return filepath.Join(cacheDir, hex.EncodeToString(sum[:8]))
}

const (
	latestCacheTTL   = 15 * time.Minute
	cacheVersionFile = ".smokeforge-version"
)

func cacheFresh(dir string, ttl time.Duration) bool {
	info, err := os.Stat(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		return false
	}

	return time.Since(info.ModTime()) < ttl
}

func cacheVersionMatches(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, cacheVersionFile)) //nolint:gosec // dir is a controlled cache path.
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(data)) == version.Current().Version
}

func writeCacheVersion(dir string) error {
	if err := os.WriteFile(
		filepath.Join(dir, cacheVersionFile),
		[]byte(version.Current().Version+"\n"),
		0o600,
	); err != nil {
		return fmt.Errorf("write cache version: %w", err)
	}

	return nil
}

func cacheUsable(opts Options, dir string) bool {
	if opts.Offline {
		return true
	}

	if opts.Tag != "latest" {
		return true
	}

	return cacheFresh(dir, latestCacheTTL) && cacheVersionMatches(dir)
}
