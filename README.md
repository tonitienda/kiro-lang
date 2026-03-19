# kiro-lang

Kiro is an **experimental, Go-backed server-side programming language** optimized for **LLM-friendly generation, repair, and review**.

Kiro deliberately keeps the surface area small:

- **one obvious syntax** for each major construct
- **explicit semantics** for effects, results, optionals, and concurrency
- **canonical formatting** so one AST has one printed shape
- **inspectable generated Go** so failures can always be debugged in a familiar backend
- **compatibility fixtures** that protect the intended stable core

## Why Kiro is optimized for LLMs

Kiro is trying to be easy for both humans and models to generate correctly:

- **small language core** for services and CLI tools
- **effects for operational impurity**: `!env`, `!fs`, `!io`, `!log`, `!net`, ...
- **`R[T,E]` for fallibility**, with `?` only for result propagation
- **`?T` for absence**, with `nil` only as an optional value
- **structured concurrency** with `group`, `spawn`, and `await`
- **block-only function bodies** so return flow is always explicit
- **stable project conventions** for tiny apps, services, and tests

See also:

- `docs/why_kiro_for_llms.md`
- `docs/llm_optimization_pass.md`
- `LLM_OPTIMIZATION_NOTES.md`

## What Kiro is for

- small HTTP/JSON services
- CLI and file-processing tools
- explicit, inspectable backend workflows
- compatibility-driven language iteration

## What Kiro is not for today

- browser/JS/WASM targets
- large framework-heavy ecosystems
- production-stability guarantees

## Status

Kiro is in an active simplification pass that treats the repository as the source of truth for the **new stable core**, not for historical syntax preservation.

Key docs:

- stable core: `docs/stable_core.md`
- design principles: `docs/design_principles.md`
- effects: `docs/effects.md`
- project layout: `docs/project_layout.md`
- service structure: `docs/service_structure.md`
- compatibility: `docs/compatibility.md`
- generated Go debugging: `docs/debugging_generated_go.md`

## Install and build Kiro itself

### From source

```bash
go build ./cmd/kiro
```

### From release artifacts

Release bundles include:

- `bin/kiro`
- `toolchain/go/...`

That keeps `kiro build`, `kiro run`, and `kiro test` usable without separately installing `go` in downstream environments.

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

```ki
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

The service template uses the canonical service shape:

- `main.ki` for wiring and effect boundaries
- `internal/config` for environment loading
- `app` for request handlers
- `test/` for handler-level tests

## Canonical language rules

### Functions use block bodies only

```ki
fn add(a:i32, b:i32) -> i32 {
  return a + b
}
```

Expression-bodied functions were removed to make control flow explicit and formatter output canonical.

### Effects are operational, not fallibility

```ki
fn load() -> R[Config, str] !env {
  let port = env.get_or("PORT", ":8080")
  return Ok(Config{port:port})
}
```

`json.encode`, `json.decode`, and `parse.i32` are pure even though they return `R[...]`.

### Result, optional, and absence stay separate

```ki
fn parse_port(raw:str) -> R[i32, str] {
  return parse.i32(raw)
}

fn maybe_name(req:http.Req) -> ?str {
  return http.query(req, "name")
}
```

- `R[T,E]` means failure is possible
- `?` propagates an `R[T,E]`
- `?T` means the value may be absent
- `nil` belongs only in optional contexts

### Structured concurrency is explicit

```ki
group {
  let ta = spawn fetch_a()
  let tb = spawn fetch_b()
  let a = await ta
  let b = await tb
  return Ok(http.json(200, json.encode(Resp{a:a b:b})?))
}
```

`group` is the preferred pattern because task lifecycles remain local and visible.

## CLI commands

```bash
./kiro fmt <paths...>
./kiro check <entry-or-path>
./kiro build <entry-or-path> [--out <file>] [--keep-gen]
./kiro run <entry-or-path> [--keep-gen] [-- <program args...>]
./kiro test <entry-or-path> [--keep-gen]
./kiro compat [root] [--mode fmt,check,inspect]
./kiro inspect go <entry-or-path> [--out-dir <dir>]
./kiro new <hello|service>
./kiro lsp
```

## Generated Go and debugging

Kiro keeps generated Go first-class:

- `kiro inspect go <entry>` writes an inspectable backend view
- `kiro build/run/test --keep-gen` preserve the generated standalone workdir
- `docs/debugging_generated_go.md` explains when to use each path

## Docs index

### Language

- `docs/language_tour.md`
- `docs/syntax-overview.md`
- `docs/design_principles.md`
- `docs/effects.md`
- `docs/concurrency.md`
- `docs/stable_core.md`
- `docs/limitations.md`

### Projects and services

- `docs/project_layout.md`
- `docs/service_structure.md`
- `docs/http_json.md`
- `docs/config.md`
- `docs/testing.md`
- `docs/testing_services.md`
- `docs/examples.md`

### Tooling and process

- `docs/compatibility.md`
- `docs/editor_setup.md`
- `docs/debugging_generated_go.md`
- `docs/stdlib_style.md`
- `docs/why_kiro_for_llms.md`
- `docs/llm_optimization_pass.md`
- `LLM_OPTIMIZATION_NOTES.md`

## Development checks

```bash
go test ./...
go run ./cmd/kiro compat
go run ./cmd/kiro check examples/hello
go run ./cmd/kiro inspect go examples/hello --out-dir .kiro-gen-example
```
