# `psr` CLI v1 Design

Status: Approved
Related: [#3](https://github.com/tunnelWithAC/protobuf-schema-registry/issues/3) (CLI tool: build and locally cache protobuf packages), [#4](https://github.com/tunnelWithAC/protobuf-schema-registry/pull/4) (draft `psr.toml` manifest schema), [#5](https://github.com/tunnelWithAC/protobuf-schema-registry/issues/5) (follow-up: `psr.lock`)

## Problem

Issue #3 asks for a CLI that resolves protobuf package dependencies and builds/generates code from them, with local caching so builds can go offline once packages are cached. The registry server itself doesn't exist yet (out of scope for #3, tracked separately), so there is nothing real for a registry client to talk to.

## Scope decisions for v1

- **No registry client.** Dependencies are local path references only (`{ path = "../foo" }`). A registry-backed dependency resolver is future work once the registry server exists.
- **No lockfile.** Path deps have no version to pin, so there's nothing meaningful to lock. Tracked separately in [#5](https://github.com/tunnelWithAC/protobuf-schema-registry/issues/5) — add `psr.lock` once versioned registry deps exist.
- **Codegen is in scope**, using [`bufbuild/protocompile`](https://github.com/bufbuild/protocompile) in-process to parse `.proto` files into a `CodeGeneratorRequest`, then shelling out to per-language plugin binaries. Targets: Go (`protoc-gen-go`, `protoc-gen-go-grpc`) and Python (`protoc-gen-python`).
- **No toolchain auto-download.** Plugin binaries must already be on `$PATH` (installed via `go install .../protoc-gen-go@version`, `pip install ...`, etc). The `[toolchain]` table in `psr.toml` is parsed but not enforced in v1 — no version-mismatch checking, no binary fetching.
- **Repo layout:** CLI lives in a new `cli/` subdirectory with its own `go.mod`, keeping the repo root free for a future registry server that may live in a different location or language.

## Manifest (`psr.toml`)

Builds on the draft in PR #4, adding a `proto_dir` field (missing from the draft) so the tool knows what to compile for the package's own sources:

```toml
[package]
name        = "example-service"   # optional — only needed if this repo also publishes a package
version     = "0.1.0"
proto_dir   = "proto"             # directory containing this package's own .proto files; default "proto"

[registries]
default = "https://registry.protobuf-schema-registry.dev"  # parsed, unused in v1

[dependencies]
"acme/common" = { path = "../local-proto" }   # v1 supports path deps only

[toolchain]
protoc = "27.2"   # parsed, unused in v1 (informational)

[toolchain.plugins.go]
plugin  = "protoc-gen-go"
version = "1.34.2"   # parsed, unused in v1 (informational)

[[generate]]
language = "go"
out      = "gen/go"
plugins  = ["go", "go-grpc"]
options  = { paths = "source_relative" }

[[generate]]
language = "python"
out      = "gen/python"
plugins  = ["python"]
```

Any `[dependencies]` entry that isn't `{ path = ... }` (e.g. a bare semver string like `"^1.2"`) is a validation error in v1 — clearly message that only path deps are supported until the registry client lands.

## Architecture

```
cli/
  go.mod
  cmd/psr/main.go          — subcommand dispatch (install/build/clean); manual dispatch, no cobra
  internal/manifest/        — parse + validate psr.toml
  internal/cache/           — cache dir resolution, populate/read
  internal/resolve/         — walk [dependencies], resolve path deps into cache
  internal/codegen/         — protocompile invocation + plugin subprocess invocation
  internal/toolchain/       — locate protoc-gen-<lang> binaries on PATH
```

Cobra is deliberately skipped: three subcommands and a handful of flags don't need a CLI framework dependency.

## Commands

### `psr install`

1. Load and validate `psr.toml` from the current directory.
2. For each `[dependencies]` entry, resolve the `path` (relative to `psr.toml`'s directory), verify it exists and contains at least one `.proto` file.
3. Copy the resolved directory's `.proto` files into `~/.cache/psr/packages/<dep-name>/local/` (cache root overridable via `PSR_CACHE_DIR`).
4. Self-heal: v1 doesn't trust cache freshness — `install` always re-copies from source on every run. Path deps are cheap to re-copy and there's no immutable version to key staleness off of yet. (This keeps the door open for the future registry cache to slot into the same `internal/cache` interface, where self-heal means re-download instead of re-copy.)

### `psr build`

1. Run the equivalent of `install`.
2. For each `[[generate]]` block:
   - Collect import paths: the package's own `proto_dir` plus every resolved dependency's cached directory.
   - Use `protocompile` to parse the relevant `.proto` files into a `CodeGeneratorRequest`.
   - For each plugin name in `plugins`, locate `protoc-gen-<plugin>` on `$PATH` and pipe the request to it over stdin.
   - Write the plugin's `CodeGeneratorResponse` output to a temp directory; on full success for that block, move it into place at `out`. On any failure, leave `out` untouched (no partial writes).
3. Missing plugin binary produces an error naming the exact install command (e.g. `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`).

### `psr clean`

Removes the local cache directory (`~/.cache/psr/packages/` or `$PSR_CACHE_DIR`) and every `out` directory referenced in `[[generate]]`.

## Error handling

Fail fast with actionable, specific messages:
- Missing `psr.toml` in the working directory.
- Missing or invalid dependency `path`, or a path with no `.proto` files.
- Missing `proto_dir`.
- Missing plugin binary on `$PATH`.
- `protocompile` parse errors surfaced with file:line from the parser.

No partial/corrupt state is left on disk — codegen output is written to a temp location and only moved into `out` on success.

## Testing

- Unit tests for `internal/manifest` (parsing, validation — including the "path deps only" rejection case) and `internal/resolve`/`internal/cache` (path resolution, cache population, re-copy behavior) using `t.TempDir()` fixtures.
- Integration test for `internal/codegen`: skipped when `protoc-gen-go` isn't present on `$PATH` in the test environment; when present, runs a real fixture `.proto` through the full pipeline (protocompile → plugin → written output) and asserts the generated Go file compiles.

## Out of scope (tracked separately or deferred)

- Registry HTTP client and semver-range dependency resolution.
- `psr.lock` ([#5](https://github.com/tunnelWithAC/protobuf-schema-registry/issues/5)).
- Toolchain binary auto-download / version enforcement.
- Registry server / storage backend (tracked separately from #3).
- Workspaces, registry auth — open questions noted in PR #4, not addressed here.
