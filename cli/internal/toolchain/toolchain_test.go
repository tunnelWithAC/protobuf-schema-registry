package toolchain

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBinaryName(t *testing.T) {
	if got := BinaryName("go"); got != "protoc-gen-go" {
		t.Errorf("BinaryName(%q) = %q, want %q", "go", got, "protoc-gen-go")
	}
}

func TestFind_NotOnPath(t *testing.T) {
	if _, err := Find("definitely-not-a-real-plugin"); err == nil {
		t.Fatal("Find() error = nil, want error for plugin not on PATH")
	}
}

func TestFind_OnPath(t *testing.T) {
	binDir := t.TempDir()
	name := "protoc-gen-fake-test-plugin"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	fakeBin := filepath.Join(binDir, name)
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("writing fake plugin binary: %v", err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	path, err := Find("fake-test-plugin")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if path == "" {
		t.Error("Find() returned empty path")
	}
}
