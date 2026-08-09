package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, dir, contents string) string {
	t.Helper()
	path := filepath.Join(dir, "psr.toml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	return path
}

func TestLoad_ValidManifest(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[package]
name = "example-service"
version = "0.1.0"
proto_dir = "proto"

[dependencies]
"acme/common" = { path = "../local-proto" }

[[generate]]
language = "go"
out = "gen/go"
plugins = ["go", "go-grpc"]
options = { paths = "source_relative" }
`)

	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if m.Package.Name != "example-service" {
		t.Errorf("Package.Name = %q, want %q", m.Package.Name, "example-service")
	}
	if m.Package.ProtoDir != "proto" {
		t.Errorf("Package.ProtoDir = %q, want %q", m.Package.ProtoDir, "proto")
	}
	dep, ok := m.Dependencies["acme/common"]
	if !ok {
		t.Fatalf("missing dependency acme/common")
	}
	if dep.Path != "../local-proto" {
		t.Errorf("dep.Path = %q, want %q", dep.Path, "../local-proto")
	}
	if len(m.Generate) != 1 || m.Generate[0].Language != "go" {
		t.Fatalf("Generate = %+v, want one go target", m.Generate)
	}
	if m.Generate[0].Options["paths"] != "source_relative" {
		t.Errorf("Generate[0].Options[paths] = %q, want %q", m.Generate[0].Options["paths"], "source_relative")
	}
}

func TestLoad_DefaultProtoDir(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[package]
name = "example-service"
`)

	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if m.Package.ProtoDir != "proto" {
		t.Errorf("Package.ProtoDir = %q, want default %q", m.Package.ProtoDir, "proto")
	}
}

func TestLoad_RejectsVersionDependency(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[dependencies]
"acme/common" = "^1.2"
`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want error for version dependency")
	}
}

func TestLoad_RejectsMissingPath(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[dependencies]
"acme/common" = {}
`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want error for dependency missing path")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "does-not-exist.toml")); err == nil {
		t.Fatal("Load() error = nil, want error for missing file")
	}
}

func TestManifest_ProtoDir(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[package]
proto_dir = "proto"
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := filepath.Join(dir, "proto")
	if got := m.ProtoDir(); got != want {
		t.Errorf("ProtoDir() = %q, want %q", got, want)
	}
}

func TestManifest_DependencyPath(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[dependencies]
"acme/common" = { path = "../local-proto" }
`)
	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := filepath.Join(dir, "../local-proto")
	if got := m.DependencyPath(m.Dependencies["acme/common"]); got != want {
		t.Errorf("DependencyPath() = %q, want %q", got, want)
	}
}

func TestLoad_IgnoresUnknownToolchainTable(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[package]
name = "example-service"

[toolchain]
[toolchain.plugins.go]
enabled = true
`)

	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if m.Package.Name != "example-service" {
		t.Errorf("Package.Name = %q, want %q", m.Package.Name, "example-service")
	}
}

func TestLoad_RejectsMixedPathAndVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeManifest(t, dir, `
[package]
name = "example-service"

[dependencies]
"acme/common" = { path = "../local-proto", version = "1.0" }
`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want error for dependency with both path and version")
	}
}
