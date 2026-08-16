package codegen

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

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

func TestWriteFiles_RejectsInsertionPoint(t *testing.T) {
	dir := t.TempDir()
	files := []*pluginpb.CodeGeneratorResponse_File{
		{
			Name:           proto.String("greeting.pb.go"),
			InsertionPoint: proto.String("imports"),
			Content:        proto.String("// fragment"),
		},
	}
	err := writeFiles(files, dir, "go", map[string]string{})
	if err == nil {
		t.Fatal("writeFiles() error = nil, want error for file with InsertionPoint set")
	}
	if !strings.Contains(err.Error(), "insertion point") {
		t.Errorf("writeFiles() error = %v, want mention of insertion point", err)
	}
}

func TestWriteFiles_RejectsCrossPluginCollision(t *testing.T) {
	dir := t.TempDir()
	written := map[string]string{}

	goFiles := []*pluginpb.CodeGeneratorResponse_File{
		{Name: proto.String("greeting.pb.go"), Content: proto.String("// go")},
	}
	if err := writeFiles(goFiles, dir, "go", written); err != nil {
		t.Fatalf("writeFiles() first call error = %v, want nil", err)
	}

	grpcFiles := []*pluginpb.CodeGeneratorResponse_File{
		{Name: proto.String("greeting.pb.go"), Content: proto.String("// go-grpc")},
	}
	err := writeFiles(grpcFiles, dir, "go-grpc", written)
	if err == nil {
		t.Fatal("writeFiles() error = nil, want error for filename collision across plugins")
	}
	if !strings.Contains(err.Error(), "greeting.pb.go") {
		t.Errorf("writeFiles() error = %v, want mention of colliding file name", err)
	}
}

// TestSwapOutput_FinalRenameFailure_RestoresExistingOutput exercises the specific failure
// mode where the initial "move outDir aside" rename succeeds but the final
// "move tmpDir into outDir" rename then fails (e.g. tmpDir vanishing out from under us,
// or in production an EXDEV/permissions failure). It asserts swapOutput restores the
// original outDir contents rather than leaving outDir missing/empty, per the invariant
// that a failed swap must never leave outDir worse off than before the call.
//
// A permission-based simulation (chmod'ing outDir's parent read-only) cannot isolate this
// specific ordering: both renames operate on entries within the same parent directory
// with the same permission bits, so any restriction that blocks the second rename would
// also block the first, never reaching the code path under test. Instead we force the
// second rename to fail deterministically by removing tmpDir after swapOutput has already
// committed to using it, which is a legitimate (if synthetic) way to make os.Rename return
// an error at that exact point without relying on platform-specific permission quirks.
func TestSwapOutput_FinalRenameFailure_RestoresExistingOutput(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	writeFile(t, filepath.Join(outDir, "existing.txt"), "keep me")

	tmpDir := filepath.Join(dir, ".psr-gen-tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("creating tmpDir: %v", err)
	}
	// Remove tmpDir out from under swapOutput so the tmpDir -> outDir rename fails
	// with ENOENT after the outDir -> asideDir rename has already succeeded.
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatalf("removing tmpDir: %v", err)
	}

	if err := swapOutput(tmpDir, outDir); err == nil {
		t.Fatal("swapOutput() error = nil, want error for missing tmpDir")
	}

	got, err := os.ReadFile(filepath.Join(outDir, "existing.txt"))
	if err != nil {
		t.Fatalf("outDir should be restored after a failed swap, stat error = %v", err)
	}
	if string(got) != "keep me" {
		t.Errorf("restored existing.txt content = %q, want %q", got, "keep me")
	}

	asideDir := outDir + ".psr-gen-old"
	if _, err := os.Stat(asideDir); !os.IsNotExist(err) {
		t.Errorf("aside dir %q should not remain after successful restore, stat error = %v", asideDir, err)
	}
}
