package codegen

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// TestBuildRequest_ProtoFileOrderAndDedup checks the topological invariants of the
// ProtoFile list on a diamond import graph with a transitive well-known-type import:
// every transitively-reachable file appears exactly once, and every file's declared
// dependencies appear before it. It needs no external plugin binary, so it always runs.
func TestBuildRequest_ProtoFileOrderAndDedup(t *testing.T) {
	protoDir := t.TempDir()
	// Diamond: top -> {left, right} -> base -> google/protobuf/timestamp.proto
	writeFile(t, filepath.Join(protoDir, "base.proto"), `
syntax = "proto3";
package diamond;
option go_package = "example/gen/diamond";
import "google/protobuf/timestamp.proto";
message Base { google.protobuf.Timestamp at = 1; }
`)
	writeFile(t, filepath.Join(protoDir, "left.proto"), `
syntax = "proto3";
package diamond;
option go_package = "example/gen/diamond";
import "base.proto";
message Left { Base base = 1; }
`)
	writeFile(t, filepath.Join(protoDir, "right.proto"), `
syntax = "proto3";
package diamond;
option go_package = "example/gen/diamond";
import "base.proto";
message Right { Base base = 1; }
`)
	writeFile(t, filepath.Join(protoDir, "top.proto"), `
syntax = "proto3";
package diamond;
option go_package = "example/gen/diamond";
import "left.proto";
import "right.proto";
message Top { Left l = 1; Right r = 2; }
`)

	req, err := BuildRequest(context.Background(), protoDir, []string{protoDir}, "")
	if err != nil {
		t.Fatalf("BuildRequest() error = %v", err)
	}

	names := make([]string, 0, len(req.ProtoFile))
	for _, f := range req.ProtoFile {
		names = append(names, f.GetName())
	}

	// Every transitively-reachable file is present, exactly once (base and the
	// well-known timestamp.proto are each reachable via two paths).
	want := []string{
		"base.proto",
		"google/protobuf/timestamp.proto",
		"left.proto",
		"right.proto",
		"top.proto",
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(sorted, want) {
		t.Fatalf("ProtoFile names (sorted) = %v, want %v", sorted, want)
	}

	// Dependency-first ordering: each file's imports precede it.
	index := make(map[string]int, len(names))
	for i, n := range names {
		if prev, dup := index[n]; dup {
			t.Fatalf("ProtoFile contains duplicate %q at indexes %d and %d: %v", n, prev, i, names)
		}
		index[n] = i
	}
	for i, f := range req.ProtoFile {
		for _, dep := range f.GetDependency() {
			j, ok := index[dep]
			if !ok {
				t.Errorf("ProtoFile %q depends on %q which is missing from ProtoFile: %v", f.GetName(), dep, names)
				continue
			}
			if j >= i {
				t.Errorf("ProtoFile order = %v: dependency %q (index %d) must precede %q (index %d)", names, dep, j, f.GetName(), i)
			}
		}
	}

	wantGenerate := []string{"base.proto", "left.proto", "right.proto", "top.proto"}
	if !reflect.DeepEqual(req.FileToGenerate, wantGenerate) {
		t.Errorf("FileToGenerate = %v, want %v", req.FileToGenerate, wantGenerate)
	}
}

func TestBuildRequest_NoProtoFiles(t *testing.T) {
	protoDir := t.TempDir()
	if _, err := BuildRequest(context.Background(), protoDir, []string{protoDir}, ""); err == nil {
		t.Fatal("BuildRequest() error = nil, want error for empty proto dir")
	}
}
