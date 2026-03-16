# kiro-lang

Kiro is an experimental server-side language.

This repository currently contains the **Phase 1 bootstrap**:

- Go module and CLI skeleton (`kiro`)
- Lexer with line/column token metadata
- Parser for a constrained subset (`mod`, `import`, `type` structs, `fn` signatures + body capture)
- AST package
- Deterministic formatter (`kiro fmt`)
- Early tests for lexer/parser/formatter
- Example `.ki` programs

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
- `docs`: spec, syntax overview, roadmap

## Status

Current implementation aligns with early Milestone 1-2 scaffolding and keeps architecture simple for later semantic analysis, code generation, and stdlib work.
