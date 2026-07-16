# protobuf-schema-registry

An open source schema registry for Protocol Buffers (`.proto`) files.

`protobuf-schema-registry` lets teams publish `.proto` files to a central registry so they can be referenced as versioned, remote packages from other projects — similar to how npm, NuGet, or crates.io work for their respective ecosystems, but for protobuf schemas.

## Why

Protobuf schemas are often shared across many services and repositories, but there's no standard way to distribute and version them independently of the code that uses them. Teams end up copy-pasting `.proto` files, vendoring them via git submodules, or building bespoke tooling. This project aims to provide a standard, pluggable registry for publishing and consuming protobuf packages.

## Key Features

- **Publish packages** — push a set of `.proto` files to the registry as a named, versioned package.
- **Reference remote packages** — depend on published packages from other projects instead of vendoring schema files.
- **Versioning** — packages are versioned so consumers can pin to a specific release and upgrade deliberately.
- **Pluggable storage backends** — bring your own storage (e.g. filesystem, S3-compatible object storage, database) instead of being locked into one provider.

## Status

This project is in early development. The core registry API, storage backend interface, and package format are still being designed. Contributions and design discussions are welcome — see the [issues](../../issues) for ongoing design work.

## Contributing

Issues and pull requests are welcome. If you're interested in a particular area (storage backends, CLI tooling, versioning semantics, etc.), check open issues or start a discussion before submitting large changes.

## License

TBD.
