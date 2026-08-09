package resolve

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/cache"
	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/manifest"
)

// Resolved records the cache directory a dependency was resolved into.
type Resolved struct {
	Name string
	Dir  string
}

// Install resolves every dependency declared in m into the cache rooted at cacheRoot,
// copying .proto files from each dependency's source path. It always re-copies from
// source on every run rather than trusting existing cache contents (self-heal by
// re-resolution), which is safe because v1 only supports cheap-to-re-copy local paths.
func Install(m *manifest.Manifest, cacheRoot string) ([]Resolved, error) {
	names := make([]string, 0, len(m.Dependencies))
	for name := range m.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)

	resolved := make([]Resolved, 0, len(names))
	for _, name := range names {
		dep := m.Dependencies[name]
		srcDir := m.DependencyPath(dep)

		info, err := os.Stat(srcDir)
		if err != nil {
			return nil, fmt.Errorf("dependency %q: path %q: %w", name, srcDir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("dependency %q: path %q is not a directory", name, srcDir)
		}

		destDir := cache.PackageDir(cacheRoot, name)
		if err := copyProtoFiles(srcDir, destDir); err != nil {
			return nil, fmt.Errorf("dependency %q: %w", name, err)
		}

		n, err := countProtoFiles(destDir)
		if err != nil {
			return nil, fmt.Errorf("dependency %q: %w", name, err)
		}
		if n == 0 {
			return nil, fmt.Errorf("dependency %q: path %q contains no .proto files", name, srcDir)
		}

		resolved = append(resolved, Resolved{Name: name, Dir: destDir})
	}
	return resolved, nil
}

func copyProtoFiles(srcDir, destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("clearing cache dir %q: %w", destDir, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("creating cache dir %q: %w", destDir, err)
	}
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0o644)
	})
}

func countProtoFiles(dir string) (int, error) {
	count := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".proto") {
			count++
		}
		return nil
	})
	return count, err
}
