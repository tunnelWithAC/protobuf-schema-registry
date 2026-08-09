package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Dir          string
	Package      PackageConfig
	Dependencies map[string]Dependency
	Generate     []GenerateTarget
}

type PackageConfig struct {
	Name     string
	Version  string
	ProtoDir string `toml:"proto_dir"`
}

type Dependency struct {
	Path     string
	Version  string
	Registry string
}

type GenerateTarget struct {
	Language string
	Out      string
	Plugins  []string
	Options  map[string]string
}

// Load reads and validates a psr.toml manifest at path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	var m Manifest
	if _, err := toml.Decode(string(data), &m); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}

	m.Dir = filepath.Dir(path)
	if m.Package.ProtoDir == "" {
		m.Package.ProtoDir = "proto"
	}

	if err := m.validate(); err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *Manifest) validate() error {
	for name, dep := range m.Dependencies {
		if dep.Version != "" || dep.Registry != "" {
			return fmt.Errorf("dependency %q: version/registry dependencies aren't supported yet in v1, only { path = \"...\" } is supported", name)
		}
		if dep.Path == "" {
			return fmt.Errorf("dependency %q: missing required \"path\" field", name)
		}
	}
	return nil
}

// ProtoDir returns the absolute path to this package's own .proto sources.
func (m *Manifest) ProtoDir() string {
	return filepath.Join(m.Dir, m.Package.ProtoDir)
}

// DependencyPath returns the absolute path a dependency's declared path resolves to,
// relative to the manifest's own directory.
func (m *Manifest) DependencyPath(dep Dependency) string {
	if filepath.IsAbs(dep.Path) {
		return dep.Path
	}
	return filepath.Join(m.Dir, dep.Path)
}
