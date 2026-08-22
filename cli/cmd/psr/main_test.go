package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildPsr compiles the psr binary into a temp dir and returns its path.
func buildPsr(t *testing.T) string {
	t.Helper()
	binPath := filepath.Join(t.TempDir(), "psr")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building psr: %v\n%s", err, out)
	}
	return binPath
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestInstall_CLI(t *testing.T) {
	psr := buildPsr(t)

	depDir := t.TempDir()
	writeFile(t, filepath.Join(depDir, "common.proto"), `syntax = "proto3"; package acme.common;`)

	projDir := t.TempDir()
	writeFile(t, filepath.Join(projDir, "proto", "greeting.proto"), `syntax = "proto3"; package example;`)
	writeFile(t, filepath.Join(projDir, "psr.toml"), `
[package]
proto_dir = "proto"

[dependencies]
"acme/common" = { path = "`+filepath.ToSlash(depDir)+`" }
`)

	cacheDir := t.TempDir()
	cmd := exec.Command(psr, "install")
	cmd.Dir = projDir
	cmd.Env = append(os.Environ(), "PSR_CACHE_DIR="+cacheDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psr install: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(cacheDir, "acme/common", "local", "common.proto")); err != nil {
		t.Errorf("expected dependency cached, stat error = %v", err)
	}
}

func TestBuild_CLI(t *testing.T) {
	if _, err := exec.LookPath("protoc-gen-go"); err != nil {
		t.Skip("protoc-gen-go not on PATH; skipping build CLI test")
	}

	psr := buildPsr(t)

	projDir := t.TempDir()
	writeFile(t, filepath.Join(projDir, "proto", "greeting.proto"), `
syntax = "proto3";
package example;
option go_package = "example/gen";
message Greeting { string message = 1; }
`)
	writeFile(t, filepath.Join(projDir, "psr.toml"), `
[package]
proto_dir = "proto"

[[generate]]
language = "go"
out = "gen/go"
plugins = ["go"]
options = { paths = "source_relative" }
`)

	cmd := exec.Command(psr, "build")
	cmd.Dir = projDir
	cmd.Env = append(os.Environ(), "PSR_CACHE_DIR="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psr build: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(projDir, "gen", "go", "greeting.pb.go")); err != nil {
		t.Errorf("expected generated file, stat error = %v", err)
	}
}

func TestClean_CLI(t *testing.T) {
	psr := buildPsr(t)

	projDir := t.TempDir()
	writeFile(t, filepath.Join(projDir, "psr.toml"), `
[[generate]]
language = "go"
out = "gen/go"
plugins = ["go"]
`)
	writeFile(t, filepath.Join(projDir, "gen", "go", "leftover.pb.go"), "stale")

	cacheDir := t.TempDir()
	writeFile(t, filepath.Join(cacheDir, "acme/common", "local", "common.proto"), "cached")

	cmd := exec.Command(psr, "clean")
	cmd.Dir = projDir
	cmd.Env = append(os.Environ(), "PSR_CACHE_DIR="+cacheDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psr clean: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(projDir, "gen", "go")); !os.IsNotExist(err) {
		t.Errorf("gen/go should be removed, stat error = %v", err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Errorf("cache dir should be removed, stat error = %v", err)
	}
}
