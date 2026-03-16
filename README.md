# kiro-lang

Kiro is an experimental server-side language.

This repository currently contains a **Phase 4 early frontend slice** with:

- Go module and CLI skeleton (`kiro`)
- Lexer with line/column token metadata
- Parser for:
  - `mod`, `import`, module-level `const`
  - `type` structs
  - function declarations with **expression bodies** (`=`) and **block bodies** (`{ ... }`)
  - value-receiver method declarations (`fn (u:User) name(...) -> ...`)
  - import paths like `app/router`
- AST package with receiver-aware function declarations
- Deterministic formatter (`kiro fmt`) for current syntax
- Tests for lexer/parser/formatter
- Optional type references in signatures/fields (`?T`)
- Reserved lexer keywords for upcoming concurrency syntax (`spawn`, `await`)
- Example `.ki` programs for Phase 2, Phase 3, and initial Phase 4 demos

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
- `PHASE4_NOTES.md`: Phase 4 updates, limitations, and next-scope recommendations

## Status

Kiro is still intentionally small. Current work focuses on incremental, readable parser/formatter evolution and staged language-surface additions before full semantic/codegen/runtime milestones.
