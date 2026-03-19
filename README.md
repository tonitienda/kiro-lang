# kiro-lang

Kiro is an **experimental, Go-backed server-side programming language**.

It is designed for small services, CLI tools, and pragmatic backend workflows where generated Go stays inspectable even when the normal workflow is `kiro check`, `kiro build`, `kiro run`, and `kiro test`.

## What Kiro is for

- small HTTP/JSON services
- CLI and file-processing tools
- explicit result/optional flow (`R[T,E]`, `?`, `?T`)
- explicit operational effects on function signatures (`!env`, `!fs`, `!io`, `!log`, `!net`, ...)
- deterministic formatting and compatibility-driven development

## What Kiro is not for (today)

- browser/JS/WASM targets
- large framework-heavy ecosystems
- production-stability guarantees

## Status

Kiro is in **Phase 12: standalone CLI toolchain and release packaging**.

- stable experimental core: `docs/stable_core.md`
- release/runtime notes: `RELEASE_TOOLCHAIN_NOTES.md`
- limitations: `docs/limitations.md`
- compatibility policy: `docs/compatibility.md`

## Install and build Kiro itself

### From source

```bash
go build ./cmd/kiro
```

Building Kiro from source still requires a local Go installation.

### From release artifacts

Experimental release bundles are built for:

- Linux amd64
- Linux arm64
- macOS amd64
- macOS arm64

Each release archive contains:

- `bin/kiro`
- `toolchain/go/...` (a bundled Go toolchain used internally by Kiro)
- release notes pointers

A user running the released `kiro` binary does **not** need to install the `go` CLI separately for normal Kiro workflows.

## Quick start

```bash
./kiro new hello
./kiro fmt hello
./kiro check hello
./kiro build hello --out ./hello-bin
./hello-bin
./kiro inspect go hello --out-dir hello/.kiro-gen
```

## Hello world

`kiro new hello` scaffolds:

```kiro
mod main

fn main() -> i32 !io {
  println("hello")
  return 0
}
```

## Tiny service example

```bash
./kiro new service
./kiro check service
./kiro test service
PORT=:8080 ./kiro run service
```

The service template includes:

- `internal/config` for explicit config loading
- `app` module for handlers
- handler-level tests under `test/`
- explicit function effect declarations for environment, logging, networking, and I/O boundaries

## CLI commands

Canonical commands:

```bash
./kiro fmt <paths...>
./kiro check <entry-or-path>
./kiro build <entry-or-path> [--out <file>] [--keep-gen]
./kiro run <entry-or-path> [--keep-gen] [-- <program args...>]
./kiro test <entry-or-path> [--keep-gen]
./kiro compat [root] [--mode fmt,check,inspect]
./kiro inspect go <entry-or-path> [--out-dir <dir>]
./kiro new <hello|service>
```

### Runtime command notes

- `kiro build` lowers a Kiro project into a generated Go work directory and produces a native executable.
- `kiro run` uses the same machinery, then executes the produced program and forwards exit codes.
- `kiro test` discovers `fn test_*()` functions and runs them through the same bundled toolchain path.
- `--keep-gen` preserves the generated Go work directory used by `build`, `run`, or `test` for debugging.
- `kiro inspect go` remains the explicit source-to-Go inspection workflow for understanding the compiler output.

## Standalone toolchain model

Kiro keeps Go as the backend, but release bundles carry their own Go toolchain under `toolchain/go`.

At runtime the CLI looks for a Go compiler in this order:

1. `KIRO_GO_BIN`
2. `KIRO_TOOLCHAIN_DIR/go/bin/go`
3. `toolchain/go/bin/go` relative to the `kiro` executable
4. `go` on `PATH` (developer/source-build fallback)

That means downstream repositories can install a released Kiro archive and run:

```bash
kiro check .
kiro build .
kiro test .
```

without installing Go separately.

## Generated Go and debugging

Kiro keeps generated Go first-class:

- `kiro inspect go <entry>` writes the inspectable backend view into `.kiro-gen`.
- `kiro build/run/test --keep-gen` preserve the standalone execution work directory that was compiled into the final binary.
- `docs/debugging_generated_go.md` explains when to use each path.

## Project structure (high level)

- `cmd/kiro`: CLI entrypoint
- `internal/*`: compiler, formatter, project loader, build orchestration, toolchain lookup, compat runner
- `compat/`: compatibility fixtures
- `examples/`: example projects and test shapes
- `docs/`: language, workflow, compatibility, editor, and release docs
- `scripts/`: release packaging and standalone validation scripts

## Docs index

- Language and style
  - `docs/language_tour.md`
  - `docs/design_principles.md`
  - `docs/effects.md`
  - `docs/stable_core.md`
  - `docs/stability.md`
  - `docs/limitations.md`
- Projects, services, and testing
  - `docs/project_layout.md`
  - `docs/service_structure.md`
  - `docs/http_json.md`
  - `docs/config.md`
  - `docs/testing.md`
  - `docs/testing_services.md`
- Tooling and process
  - `docs/debugging_generated_go.md`
  - `docs/examples.md`
  - `docs/releasing.md`
  - `docs/contributing.md`
  - `docs/editor_setup.md`
- History/notes
  - `RELEASE_TOOLCHAIN_NOTES.md`
  - `PHASE10_NOTES.md`
  - `PHASE11_NOTES.md`

## Development checks

```bash
go test ./...
go run ./cmd/kiro compat
./scripts/package_release.sh dev 1.22.12 linux amd64
./scripts/verify_release_bundle.sh dist/kiro-dev-linux-amd64.tar.gz
```
