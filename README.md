# kiro-lang

Kiro is an **experimental, Go-backed server-side programming language**.

It is designed for small services, CLI tools, and pragmatic backend workflows where generated Go is inspectable.

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

Kiro is in **Phase 11: editor tooling and language-server support**.

- stable experimental core: `docs/stable_core.md`
- limitations: `docs/limitations.md`
- compatibility policy: `docs/compatibility.md`

## Build from source

```bash
go build ./cmd/kiro
```

This produces a `kiro` binary in the current directory.

## Quick start

```bash
./kiro new hello
./kiro fmt hello
./kiro check hello
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
./kiro fmt service
./kiro check service
./kiro inspect go service --out-dir service/.kiro-gen
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
./kiro compat [root] [--mode fmt,check,inspect]
./kiro inspect go <entry-or-path> [--out-dir <dir>]
./kiro new <hello|service>
```

Additional command stubs currently return placeholder messages in this repository slice:

```bash
./kiro build <entry>
./kiro run <entry>
./kiro test <path>
```

Use `./kiro help` for command help.

## Project structure (high level)

- `cmd/kiro`: CLI entrypoint
- `internal/*`: compiler, formatter, project loader, codegen, compat runner
- `compat/`: compatibility fixtures
- `examples/`: runnable syntax/service/testing examples
- `docs/`: language, workflow, compatibility, and release docs

## Docs index

- Language and style
  - `docs/language_tour.md`
  - `docs/design_principles.md`
  - `docs/effects.md`
  - `docs/stable_core.md`
  - `docs/stability.md`
  - `docs/limitations.md`
- Projects and services
  - `docs/project_layout.md`
  - `docs/service_structure.md`
  - `docs/http_json.md`
  - `docs/config.md`
  - `docs/concurrency.md`
- Testing and compatibility
  - `docs/testing.md`
  - `docs/testing_services.md`
  - `docs/compatibility.md`
- Tooling and process
  - `docs/debugging_generated_go.md`
  - `docs/examples.md`
  - `docs/releasing.md`
  - `docs/contributing.md`
  - `docs/editor_setup.md`
- History/notes
  - `CHANGELOG.md`
  - `PHASE10_NOTES.md`
  - `PHASE11_NOTES.md`

## Development checks

```bash
go test ./...
go run ./cmd/kiro compat
```
