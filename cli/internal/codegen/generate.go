package codegen

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/manifest"
	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/toolchain"
)

// RunPlugin sends req to the protoc-gen-<plugin> binary found on PATH and returns its
// parsed response. It returns an error if the binary is missing, exits non-zero, or
// reports a generation error via CodeGeneratorResponse.Error.
func RunPlugin(ctx context.Context, plugin string, req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	binPath, err := toolchain.Find(plugin)
	if err != nil {
		return nil, err
	}

	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request for plugin %q: %w", plugin, err)
	}

	cmd := exec.CommandContext(ctx, binPath)
	cmd.Stdin = bytes.NewReader(reqBytes)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running plugin %q: %w (stderr: %s)", plugin, err, stderr.String())
	}

	var resp pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parsing response from plugin %q: %w", plugin, err)
	}
	if resp.Error != nil && *resp.Error != "" {
		return nil, fmt.Errorf("plugin %q reported error: %s", plugin, *resp.Error)
	}

	return &resp, nil
}

// Generate compiles target's language sources and runs every plugin listed in
// target.Plugins, writing their combined output into target.Out (relative to m.Dir).
// Output is staged in a temp directory and only moved into place once every plugin has
// succeeded, so a failed generate leaves any existing out directory untouched.
func Generate(ctx context.Context, m *manifest.Manifest, target manifest.GenerateTarget, importPaths []string) error {
	parameter := encodeParameter(target.Options)

	req, err := BuildRequest(ctx, m.ProtoDir(), importPaths, parameter)
	if err != nil {
		return fmt.Errorf("generate[%s]: %w", target.Language, err)
	}

	outDir := filepath.Join(m.Dir, target.Out)
	if err := os.MkdirAll(filepath.Dir(outDir), 0o755); err != nil {
		return fmt.Errorf("generate[%s]: creating parent of output dir %q: %w", target.Language, outDir, err)
	}
	tmpDir, err := os.MkdirTemp(filepath.Dir(outDir), ".psr-gen-*")
	if err != nil {
		return fmt.Errorf("generate[%s]: creating temp output dir: %w", target.Language, err)
	}
	defer os.RemoveAll(tmpDir)

	for _, plugin := range target.Plugins {
		resp, err := RunPlugin(ctx, plugin, req)
		if err != nil {
			return fmt.Errorf("generate[%s]: %w", target.Language, err)
		}
		if err := writeFiles(resp.File, tmpDir); err != nil {
			return fmt.Errorf("generate[%s]: %w", target.Language, err)
		}
	}

	if err := os.RemoveAll(outDir); err != nil {
		return fmt.Errorf("generate[%s]: clearing output dir %q: %w", target.Language, outDir, err)
	}
	if err := os.Rename(tmpDir, outDir); err != nil {
		return fmt.Errorf("generate[%s]: moving generated output into %q: %w", target.Language, outDir, err)
	}
	return nil
}

func writeFiles(files []*pluginpb.CodeGeneratorResponse_File, destDir string) error {
	for _, f := range files {
		if f.Name == nil {
			continue
		}
		destPath := filepath.Join(destDir, f.GetName())
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("creating output dir for %q: %w", f.GetName(), err)
		}
		if err := os.WriteFile(destPath, []byte(f.GetContent()), 0o644); err != nil {
			return fmt.Errorf("writing generated file %q: %w", f.GetName(), err)
		}
	}
	return nil
}
