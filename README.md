# kiro-lang

Kiro is an experimental server-side language.

This repository currently contains a **Phase 2-in-progress frontend** with:

- Go module and CLI skeleton (`kiro`)
- Lexer with line/column token metadata
- Parser for:
  - `mod`, `import`
  - `type` structs
  - function declarations with **expression bodies** (`=`) and **block bodies** (`{ ... }`)
  - import paths like `app/router`
- AST package
- Deterministic formatter (`kiro fmt`) for current syntax
- Tests for lexer/parser/formatter
- Example `.ki` programs for planned Phase 2 demos

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
- `PHASE2_NOTES.md`: implemented pieces, limits, next steps

## Status

Kiro is still intentionally small. Current work focuses on incremental, readable parser/formatter evolution before semantic analysis and code generation milestones.
