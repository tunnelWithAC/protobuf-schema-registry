package manifest

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

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
	md, err := toml.Decode(string(data), &m)
	if err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	if err := checkUndecodedKeys(md); err != nil {
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

// checkUndecodedKeys rejects any TOML key that BurntSushi's decoder left undecoded
// (i.e. doesn't map to a Manifest struct field) unless it falls under an explicit
// allowlist. This catches typo'd keys (e.g. "protodir" instead of "proto_dir") that
// would otherwise be silently ignored, while still permitting the [toolchain] table
// (parsed-but-ignored per the v1 spec) and the decorative package.description field.
func checkUndecodedKeys(md toml.MetaData) error {
	for _, key := range md.Undecoded() {
		k := key.String()
		if k == "toolchain" || strings.HasPrefix(k, "toolchain.") {
			continue
		}
		if k == "package.description" {
			continue
		}
		return fmt.Errorf("unknown key %q", k)
	}
	return nil
}

func (m *Manifest) validate() error {
	for name, dep := range m.Dependencies {
		if name != path.Clean(name) || path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, `\`) {
			return fmt.Errorf("dependency %q: name must be a clean relative path (no \"..\", no absolute paths)", name)
		}
		if dep.Version != "" || dep.Registry != "" {
			return fmt.Errorf("dependency %q: version/registry dependencies aren't supported yet in v1, only { path = \"...\" } is supported", name)
		}
		if dep.Path == "" {
			return fmt.Errorf("dependency %q: missing required \"path\" field", name)
		}
	}

	for i, g := range m.Generate {
		if g.Out == "" {
			return fmt.Errorf("generate[%d]: missing required \"out\" field", i)
		}
		if filepath.IsAbs(g.Out) || !filepath.IsLocal(g.Out) {
			return fmt.Errorf("generate[%d]: out %q must be a relative path inside the package directory", i, g.Out)
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
