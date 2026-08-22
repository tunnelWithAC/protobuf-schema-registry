package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/cache"
	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/codegen"
	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/manifest"
	"github.com/tunnelWithAC/protobuf-schema-registry/cli/internal/resolve"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: psr <install|build|clean>")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "install":
		err = runInstall()
	case "build":
		err = runBuild()
	case "clean":
		err = runClean()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "psr: "+err.Error())
		os.Exit(1)
	}
}

func loadManifest() (*manifest.Manifest, error) {
	return manifest.Load("psr.toml")
}

func runInstall() error {
	m, err := loadManifest()
	if err != nil {
		return err
	}
	root, err := cache.Root()
	if err != nil {
		return err
	}
	resolved, err := resolve.Install(m, root)
	if err != nil {
		return err
	}
	for _, r := range resolved {
		fmt.Printf("resolved %s -> %s\n", r.Name, r.Dir)
	}
	return nil
}

func runBuild() error {
	m, err := loadManifest()
	if err != nil {
		return err
	}
	root, err := cache.Root()
	if err != nil {
		return err
	}
	resolved, err := resolve.Install(m, root)
	if err != nil {
		return err
	}

	importPaths := []string{m.ProtoDir()}
	for _, r := range resolved {
		importPaths = append(importPaths, r.Dir)
	}

	ctx := context.Background()
	for _, target := range m.Generate {
		if err := codegen.Generate(ctx, m, target, importPaths); err != nil {
			return err
		}
		fmt.Printf("generated %s -> %s\n", target.Language, target.Out)
	}
	return nil
}

func runClean() error {
	m, err := loadManifest()
	if err != nil {
		return err
	}
	root, err := cache.Root()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("removing cache dir %q: %w", root, err)
	}
	fmt.Printf("removed %s\n", root)

	for _, target := range m.Generate {
		outDir := filepath.Join(m.Dir, target.Out)
		if err := os.RemoveAll(outDir); err != nil {
			return fmt.Errorf("removing output dir %q: %w", outDir, err)
		}
		fmt.Printf("removed %s\n", outDir)
	}
	return nil
}
