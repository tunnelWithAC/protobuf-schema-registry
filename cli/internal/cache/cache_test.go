package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoot_UsesEnvVar(t *testing.T) {
	t.Setenv("PSR_CACHE_DIR", "/tmp/custom-psr-cache")

	got, err := Root()
	if err != nil {
		t.Fatalf("Root() error = %v", err)
	}
	if got != "/tmp/custom-psr-cache" {
		t.Errorf("Root() = %q, want %q", got, "/tmp/custom-psr-cache")
	}
}

func TestRoot_DefaultsToUserCacheDir(t *testing.T) {
	t.Setenv("PSR_CACHE_DIR", "")

	got, err := Root()
	if err != nil {
		t.Fatalf("Root() error = %v", err)
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("os.UserCacheDir() error = %v", err)
	}
	want := filepath.Join(userCache, "psr", "packages")
	if got != want {
		t.Errorf("Root() = %q, want %q", got, want)
	}
}

func TestPackageDir(t *testing.T) {
	got := PackageDir("/root", "acme/common")
	want := filepath.Join("/root", "acme/common", "local")
	if got != want {
		t.Errorf("PackageDir() = %q, want %q", got, want)
	}
}
