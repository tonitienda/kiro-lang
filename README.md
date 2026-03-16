# kiro-lang

Kiro is an experimental server-side language.

This repository currently contains a **Phase 3 milestone 1 frontend** with:

- Go module and CLI skeleton (`kiro`)
- Lexer with line/column token metadata
- Parser for:
  - `mod`, `import`
  - `type` structs
  - function declarations with **expression bodies** (`=`) and **block bodies** (`{ ... }`)
  - value-receiver method declarations (`fn (u:User) name(...) -> ...`)
  - import paths like `app/router`
- AST package with receiver-aware function declarations
- Deterministic formatter (`kiro fmt`) for current syntax
- Tests for lexer/parser/formatter
- Example `.ki` programs for Phase 2 and Phase 3 milestone 1 demos

## Build

```bash
go build ./cmd/kiro
```

## Commands

```bash
./kiro fmt <paths...>
./kiro build <entry>   # milestone placeholder
./kiro run <entry>     # milestone placeholder
./kiro test <path>     # milestone placeholder
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
- `examples`: sample `.ki` programs
- `docs`: specs, syntax notes, roadmap
- `PHASE2_NOTES.md`: Phase 2 implementation notes
- `PHASE3_NOTES.md`: ongoing Phase 3 implementation notes and known limitations

## Status

Kiro is still intentionally small. Current work focuses on incremental, readable parser/formatter evolution before semantic analysis and code generation milestones.
