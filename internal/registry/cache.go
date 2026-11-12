package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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
