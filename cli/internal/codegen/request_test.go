package codegen

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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

func TestFindProtoFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.proto"), "")
	writeFile(t, filepath.Join(root, "sub", "b.proto"), "")
	writeFile(t, filepath.Join(root, "README.md"), "")

	got, err := findProtoFiles(root)
	if err != nil {
		t.Fatalf("findProtoFiles() error = %v", err)
	}
	want := []string{"a.proto", filepath.ToSlash(filepath.Join("sub", "b.proto"))}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("findProtoFiles() = %v, want %v", got, want)
	}
}

func TestEncodeParameter(t *testing.T) {
	if got := encodeParameter(nil); got != "" {
		t.Errorf("encodeParameter(nil) = %q, want empty", got)
	}
	got := encodeParameter(map[string]string{"paths": "source_relative", "annotate_code": "true"})
	want := "annotate_code=true,paths=source_relative"
	if got != want {
		t.Errorf("encodeParameter() = %q, want %q", got, want)
	}
}

func TestBuildRequest(t *testing.T) {
	protoDir := t.TempDir()
	writeFile(t, filepath.Join(protoDir, "greeting.proto"), `
syntax = "proto3";
package example;
option go_package = "example/gen";

message Greeting {
  string message = 1;
}
`)

	req, err := BuildRequest(context.Background(), protoDir, []string{protoDir}, "")
	if err != nil {
		t.Fatalf("BuildRequest() error = %v", err)
	}
	if len(req.FileToGenerate) != 1 || req.FileToGenerate[0] != "greeting.proto" {
		t.Errorf("FileToGenerate = %v, want [greeting.proto]", req.FileToGenerate)
	}
	if len(req.ProtoFile) != 1 || req.ProtoFile[0].GetName() != "greeting.proto" {
		t.Errorf("ProtoFile = %v, want one entry named greeting.proto", req.ProtoFile)
	}
}

func TestBuildRequest_NoProtoFiles(t *testing.T) {
	protoDir := t.TempDir()
	if _, err := BuildRequest(context.Background(), protoDir, []string{protoDir}, ""); err == nil {
		t.Fatal("BuildRequest() error = nil, want error for empty proto dir")
	}
}
