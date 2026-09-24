// Package codegen compiles .proto source files into a CodeGeneratorRequest that
// can be handed to a protoc-gen-<lang> plugin binary.
package codegen

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// BuildRequest compiles every .proto file found under protoDir (resolved against
// importPaths, which must include protoDir) into a CodeGeneratorRequest ready to send
// to a protoc-gen-<lang> plugin binary.
func BuildRequest(ctx context.Context, protoDir string, importPaths []string, parameter string) (*pluginpb.CodeGeneratorRequest, error) {
	targetFiles, err := findProtoFiles(protoDir)
	if err != nil {
		return nil, fmt.Errorf("finding .proto files in %q: %w", protoDir, err)
	}
	if len(targetFiles) == 0 {
		return nil, fmt.Errorf("no .proto files found in %q", protoDir)
	}

	compiler := protocompile.Compiler{
		Resolver:       protocompile.WithStandardImports(&protocompile.SourceResolver{ImportPaths: importPaths}),
		SourceInfoMode: protocompile.SourceInfoStandard,
	}

	compiled, err := compiler.Compile(ctx, targetFiles...)
	if err != nil {
		return nil, fmt.Errorf("compiling .proto files: %w", err)
	}

	seen := map[string]bool{}
	var order []*descriptorpb.FileDescriptorProto
	for _, fd := range compiled {
		collectFileDescriptors(fd, seen, &order)
	}

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: targetFiles,
		ProtoFile:      order,
	}
	if parameter != "" {
		req.Parameter = proto.String(parameter)
	}
	return req, nil
}

// findProtoFiles returns the slash-separated paths of every .proto file under
// root, relative to root, in sorted order.
func findProtoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// encodeParameter renders plugin options as the comma-separated key=value string
// that protoc passes to plugins, with keys sorted for deterministic output.
func encodeParameter(opts map[string]string) string {
	if len(opts) == 0 {
		return ""
	}
	keys := make([]string, 0, len(opts))
	for k := range opts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, opts[k]))
	}
	return strings.Join(parts, ",")
}

// collectFileDescriptors walks fd's transitive imports depth-first, appending each file
// exactly once to order with its dependencies appearing before it.
func collectFileDescriptors(fd protoreflect.FileDescriptor, seen map[string]bool, order *[]*descriptorpb.FileDescriptorProto) {
	if seen[fd.Path()] {
		return
	}
	seen[fd.Path()] = true // mark before recursing so import cycles can't loop forever
	imports := fd.Imports()
	for i := 0; i < imports.Len(); i++ {
		collectFileDescriptors(imports.Get(i).FileDescriptor, seen, order)
	}
	*order = append(*order, protodesc.ToFileDescriptorProto(fd))
}
