package cache

import (
	"fmt"
	"os"
	"path/filepath"
)

const envCacheDir = "PSR_CACHE_DIR"

// Root returns the root directory for psr's local package cache, honoring
// PSR_CACHE_DIR if set, else defaulting to os.UserCacheDir()/psr/packages.
func Root() (string, error) {
	if dir := os.Getenv(envCacheDir); dir != "" {
		return dir, nil
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("determining user cache dir: %w", err)
	}
	return filepath.Join(userCache, "psr", "packages"), nil
}

// PackageDir returns the cache directory a dependency named name should be
// resolved into, given the cache root. "local" reflects that v1 only
// resolves dependencies from local paths.
func PackageDir(root, name string) string {
	return filepath.Join(root, name, "local")
}
