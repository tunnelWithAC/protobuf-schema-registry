package codegen

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/manifest"
)

func TestRunPlugin_MissingBinary(t *testing.T) {
	_, err := RunPlugin(context.Background(), "definitely-not-a-real-plugin", nil)
	if err == nil {
		t.Fatal("RunPlugin() error = nil, want error for missing plugin binary")
	}
}

func TestGenerate_EndToEnd(t *testing.T) {
	if _, err := exec.LookPath("protoc-gen-go"); err != nil {
		t.Skip("protoc-gen-go not on PATH; skipping end-to-end codegen test")
	}

	dir := t.TempDir()
	protoDir := filepath.Join(dir, "proto")
	writeFile(t, filepath.Join(protoDir, "greeting.proto"), `
syntax = "proto3";
package example;
option go_package = "example/gen";

message Greeting {
  string message = 1;
}
`)

	m := &manifest.Manifest{
		Dir:     dir,
		Package: manifest.PackageConfig{ProtoDir: "proto"},
	}
	target := manifest.GenerateTarget{
		Language: "go",
		Out:      "gen/go",
		Plugins:  []string{"go"},
		Options:  map[string]string{"paths": "source_relative"},
	}

	if err := Generate(context.Background(), m, target, []string{protoDir}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	generated, err := os.ReadFile(filepath.Join(dir, "gen", "go", "greeting.pb.go"))
	if err != nil {
		t.Fatalf("reading generated file: %v", err)
	}
	if len(generated) == 0 {
		t.Error("generated file is empty")
	}
}

func TestGenerate_MissingPlugin_LeavesOutDirUntouched(t *testing.T) {
	dir := t.TempDir()
	protoDir := filepath.Join(dir, "proto")
	writeFile(t, filepath.Join(protoDir, "greeting.proto"), `
syntax = "proto3";
package example;
message Greeting { string message = 1; }
`)
	outDir := filepath.Join(dir, "gen", "go")
	writeFile(t, filepath.Join(outDir, "existing.txt"), "should survive a failed build")

	m := &manifest.Manifest{
		Dir:     dir,
		Package: manifest.PackageConfig{ProtoDir: "proto"},
	}
	target := manifest.GenerateTarget{
		Language: "go",
		Out:      "gen/go",
		Plugins:  []string{"definitely-not-a-real-plugin"},
	}

	if err := Generate(context.Background(), m, target, []string{protoDir}); err == nil {
		t.Fatal("Generate() error = nil, want error for missing plugin binary")
	}

	if _, err := os.Stat(filepath.Join(outDir, "existing.txt")); err != nil {
		t.Errorf("existing.txt should survive a failed Generate(), stat error = %v", err)
	}
}
