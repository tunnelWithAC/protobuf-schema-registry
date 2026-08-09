package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/manifest"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func testManifest(t *testing.T, depPath string) *manifest.Manifest {
	t.Helper()
	dir := t.TempDir()
	return &manifest.Manifest{
		Dir: dir,
		Dependencies: map[string]manifest.Dependency{
			"acme/common": {Path: depPath},
		},
	}
}

func TestInstall_CopiesProtoFiles(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "common.proto"), `syntax = "proto3"; package acme.common;`)

	m := testManifest(t, srcDir)
	cacheRoot := t.TempDir()

	resolved, err := Install(m, cacheRoot)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(resolved) != 1 || resolved[0].Name != "acme/common" {
		t.Fatalf("Install() = %+v, want one resolved dep named acme/common", resolved)
	}

	got, err := os.ReadFile(filepath.Join(resolved[0].Dir, "common.proto"))
	if err != nil {
		t.Fatalf("reading cached file: %v", err)
	}
	if string(got) != `syntax = "proto3"; package acme.common;` {
		t.Errorf("cached file contents = %q, want source contents", got)
	}
}

func TestInstall_MissingPath(t *testing.T) {
	m := testManifest(t, filepath.Join(t.TempDir(), "does-not-exist"))
	if _, err := Install(m, t.TempDir()); err == nil {
		t.Fatal("Install() error = nil, want error for missing dependency path")
	}
}

func TestInstall_NoProtoFiles(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "README.md"), "not a proto file")

	m := testManifest(t, srcDir)
	if _, err := Install(m, t.TempDir()); err == nil {
		t.Fatal("Install() error = nil, want error for dependency with no .proto files")
	}
}

func TestInstall_ReCopiesOnEachRun(t *testing.T) {
	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "common.proto"), `syntax = "proto3";`)

	m := testManifest(t, srcDir)
	cacheRoot := t.TempDir()

	resolved, err := Install(m, cacheRoot)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	staleFile := filepath.Join(resolved[0].Dir, "stale.proto")
	writeFile(t, staleFile, "stale content that shouldn't survive a re-run")

	if _, err := Install(m, cacheRoot); err != nil {
		t.Fatalf("second Install() error = %v", err)
	}
	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Errorf("stale.proto still exists after re-install, want it removed")
	}
}
