package toolchain

import (
	"fmt"
	"os/exec"
)

// BinaryName returns the expected plugin binary name for a plugin key used in
// psr.toml's [[generate]].plugins list, e.g. "go" -> "protoc-gen-go".
func BinaryName(plugin string) string {
	return "protoc-gen-" + plugin
}

// Find locates the binary for a plugin key on $PATH. v1 requires plugin binaries to
// already be installed; it does not download or version-check them.
func Find(plugin string) (string, error) {
	bin := BinaryName(plugin)
	path, err := exec.LookPath(bin)
	if err != nil {
		return "", fmt.Errorf(
			"plugin binary %q not found on PATH (needed for plugin %q): install it first, e.g. `go install google.golang.org/protobuf/cmd/%s@latest`",
			bin, plugin, bin,
		)
	}
	return path, nil
}
