# kiro-lang

Kiro is an experimental, Go-backed server-side language.

This repository now contains a **Phase 6 polish slice** focused on coherent project layout, faster validation, inspectable generated Go, and starter scaffolding.

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
  - `kiro check`
  - `kiro inspect go`
  - `kiro new`
  - `kiro build` (placeholder)
  - `kiro run` (placeholder)
  - `kiro test` (placeholder)
- Project loader with explicit module/import resolution rules
- Generated Go inspection output with source-to-file mapping
- Examples for service/CLI/testing/project patterns

## Build

```bash
go build ./cmd/kiro
```

## Commands

```bash
./kiro fmt <paths...>
./kiro check <entry-or-path>
./kiro inspect go <entry-or-path> [--out-dir <dir>]
./kiro new <hello|service>
./kiro build <entry>
./kiro run <entry>
./kiro test <path>
```

## Development

```bash
go test ./...
```

## Docs

- `docs/language_tour.md`
- `docs/project_layout.md`
- `docs/testing.md`
- `docs/http_json.md`
- `PHASE6_NOTES.md`

## Compatibility / roadmap

Kiro is still experimental.

- Stable enough today: parser/formatter workflow, project loading rules, CLI check/inspect/new behavior.
- Likely to change soon: semantic/type system implementation details, executable backend code generation, stdlib runtime APIs.
- Near-term roadmap: complete semantic checking, real codegen pipeline, practical test runner output, stdlib surface hardening.
