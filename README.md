# kiro-lang

Kiro is an **experimental Go-backed language** for **LLM-generated, LLM-reviewed, and LLM-maintained code**.

The repository now treats **aggressive simplification** as a language feature, not a migration inconvenience.

## Why Kiro is optimized for LLMs

Kiro is designed so a model can:

- generate code in **one obvious syntax**
- recover from diagnostics with **small, local edits**
- distinguish **effects**, **fallibility**, **optionality**, and **concurrency** without guesswork
- rely on **canonical formatting** as part of the language definition
- inspect generated Go when debugging or reviewing lowered code

Kiro optimizes for:

1. **predictability**
2. **regularity**
3. **semantic clarity**
4. **repairability from compiler diagnostics**
5. **canonical formatting**
6. **low ambiguity**
7. **small explicit surface area**
8. **high semantic information per token**

Read the rationale in `docs/why_kiro_for_llms.md` and the redesign record in `LLM_REDESIGN_NOTES.md`.

## Core language contract

Kiro now teaches one canonical style:

- **block-only functions** with explicit `return`
- **explicit effects** for operational impurity only
- **`R[T,E]`** for fallibility
- **`?`** only for propagating `R[T,E]`
- **`?T`** for optional values
- **`nil`** only where an optional or `nil` return type is allowed
- **`group` + `spawn` + `await`** for visible structured concurrency
- **deterministic formatting** via `kiro fmt`

### Canonical function signature

```ki
fn handler(req:http.Req) -> R[http.Resp,str] !net {
  return Ok(http.not_found())
}
```

The signature carries the information an LLM most often needs to reason correctly about behavior:

- inputs
- return type
- failure channel
- operational impurity

## Effects are operational, not fallibility

Built-in effects are intentionally small:

- `!env`
- `!fs`
- `!io`
- `!log`
- `!net`
- `!panic`
- `!proc`
- `!time`

Pure transforms are **not** effects, even when they return `R[T,E]`:

- `json.encode`
- `json.decode`
- `parse.i32`

## Quick start

```bash
go build ./cmd/kiro
./kiro new hello
./kiro fmt hello
./kiro check hello
./kiro inspect go hello --out-dir hello/.kiro-gen
```

`kiro new hello` scaffolds:

```ki
mod main

fn main() -> i32 !io {
  println("hello")
  return 0
}
```

## Canonical service shape

The service template and docs now teach a single service layout:

- `main.ki` owns startup and effect boundaries
- `app/` owns request handlers
- `internal/config/` owns environment loading
- `test/` owns handler-level tests

See:

- `docs/project_layout.md`
- `docs/service_structure.md`
- `docs/testing_services.md`
- `docs/http_json.md`

## CLI commands

```bash
./kiro fmt <paths...>
./kiro check <entry-or-path>
./kiro inspect go <entry-or-path> [--out-dir <dir>]
./kiro build <entry-or-path> [--out <file>] [--keep-gen]
./kiro run <entry-or-path> [--keep-gen] [-- <program args...>]
./kiro test <entry-or-path> [--keep-gen]
./kiro compat [root] [--mode fmt,check,inspect]
./kiro new <hello|service>
./kiro lsp
```

## Compatibility and generated Go

Kiro treats both as first-class trust mechanisms:

- compatibility fixtures protect the **new canonical language**, not old accidental syntax
- `kiro inspect go` keeps lowering visible and debuggable

## Documentation map

### Language

- `docs/design_principles.md`
- `docs/stable_core.md`
- `docs/language_tour.md`
- `docs/effects.md`
- `docs/stdlib_style.md`

### Projects

- `docs/project_layout.md`
- `docs/service_structure.md`
- `docs/testing.md`
- `docs/testing_services.md`
- `docs/http_json.md`

### Rationale and process

- `docs/why_kiro_for_llms.md`
- `docs/compatibility.md`
- `docs/migration.md`
- `LLM_REDESIGN_NOTES.md`

## Development checks

```bash
go test ./...
go run ./cmd/kiro compat
go run ./cmd/kiro check examples/hello
go run ./cmd/kiro inspect go examples/hello --out-dir .kiro-gen-example
```
