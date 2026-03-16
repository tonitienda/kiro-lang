# kiro-lang

Kiro is an experimental, Go-backed server-side language.

This repository now contains a **Phase 5 frontend slice** focused on small service-language ergonomics with deterministic tooling.

## Implemented language/tooling surface (current repo)

- Lexer with line/column token metadata
- Parser for:
  - `mod`, `import`, module-level `const`
  - `type` structs
  - function declarations with expression (`=`) and block (`{ ... }`) bodies
  - value-receiver methods (`fn (u:User) ...`)
  - optional type references in signatures/fields (`?T`)
  - top-level doc comments (`/// ...`) attached to declarations
- AST package with receiver-aware declarations and doc comment capture
- Deterministic formatter (`kiro fmt`) that preserves canonical doc comment placement
- CLI command surface:
  - `kiro fmt`
  - `kiro build` (placeholder)
  - `kiro run` (placeholder)
  - `kiro test` (placeholder)
- Reserved lexer keywords for concurrency/control-flow evolution including `spawn`, `await`, and `group`
- Examples for Phase 4 and new Phase 5 acceptance-style programs

> Note: semantic analysis, codegen, runtime helpers, and stdlib execution behavior described in roadmap notes are still incremental and not all represented in this frontend-focused slice.

## Build

```bash
go build ./cmd/kiro
```

## Commands

```bash
./kiro fmt <paths...>
./kiro build <entry>
./kiro run <entry>
./kiro test <path>
```

## Development

```bash
go test ./...
```

## Project layout

- `cmd/kiro`: CLI entrypoint
- `internal/lexer`: tokenizer
- `internal/parser`: parser
- `internal/ast`: AST nodes
- `internal/format`: canonical formatter
- `internal/cli`: command wiring
- `examples`: sample `.ki` programs, including new Phase 5 examples
- `docs`: specs, syntax notes, roadmap
- `PHASE2_NOTES.md`, `PHASE3_NOTES.md`, `PHASE4_NOTES.md`, `PHASE5_NOTES.md`: phased implementation notes

## Philosophy

Kiro aims to be a tiny, opinionated language for backend services and operational tooling:

- small syntax surface
- deterministic formatting/parsing behavior
- pragmatic Go integration
- no giant-language ambitions (no macro/typeclass/ownership complexity)
