package codegen

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"
)

// TestBuildRequest_AcceptedByPlugin checks that the CodeGeneratorRequest we build is
// actually well-formed by feeding it to a real protoc-gen-go, including a transitive
// well-known-type import. Skipped when protoc-gen-go is not installed.
func TestBuildRequest_AcceptedByPlugin(t *testing.T) {
	plugin, err := exec.LookPath("protoc-gen-go")
	if err != nil {
		t.Skip("protoc-gen-go not on PATH")
	}

	protoDir := t.TempDir()
	writeFile(t, filepath.Join(protoDir, "common", "ts.proto"), `
syntax = "proto3";
package common;
option go_package = "example/gen/common";
import "google/protobuf/timestamp.proto";
message Stamp { google.protobuf.Timestamp at = 1; }
`)
	writeFile(t, filepath.Join(protoDir, "greeting.proto"), `
syntax = "proto3";
package example;
option go_package = "example/gen";
import "common/ts.proto";
message Greeting { string message = 1; common.Stamp when = 2; }
`)

	req, err := BuildRequest(context.Background(), protoDir, []string{protoDir}, encodeParameter(map[string]string{"paths": "source_relative"}))
	if err != nil {
		t.Fatalf("BuildRequest() error = %v", err)
	}

	// Dependencies must precede dependents, and well-known imports must be included.
	var names []string
	for _, f := range req.ProtoFile {
		names = append(names, f.GetName())
	}
	want := []string{"google/protobuf/timestamp.proto", "common/ts.proto", "greeting.proto"}
	if len(names) != len(want) {
		t.Fatalf("ProtoFile = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("ProtoFile = %v, want %v", names, want)
		}
	}

	in, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	cmd := exec.Command(plugin)
	cmd.Stdin = bytes.NewReader(in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("protoc-gen-go failed: %v, stderr=%s", err, stderr.String())
	}
	var resp pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("plugin reported error: %s", resp.GetError())
	}
	if len(resp.File) != 2 {
		t.Fatalf("generated %d files, want 2", len(resp.File))
	}
}
