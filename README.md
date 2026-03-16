# kiro-lang

Kiro is an experimental, Go-backed server-side language.

This repository now contains a **Phase 9 quality/consistency slice** focused on API coherence, diagnostics clarity, compatibility discipline, and maintainable workflows.

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
  - `kiro compat`
  - `kiro inspect go`
  - `kiro new`
  - `kiro build` (placeholder)
  - `kiro run` (placeholder)
  - `kiro test` (placeholder)
- Project loader with explicit module/import resolution rules
- Generated Go inspection output with source partitioning (`src/` and `runtime/`) and declaration-origin comments
- `kiro new service` template aligned to config + handler + test layout
- Examples for service/CLI/testing/project patterns

## Build

```bash
go build ./cmd/kiro
```

## Commands

```bash
./kiro fmt <paths...>
./kiro check <entry-or-path>
./kiro compat [root] [--mode fmt,check,inspect]
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
- `docs/packages.md`
- `docs/service_structure.md`
- `docs/config.md`
- `docs/http_json.md`
- `docs/testing.md`
- `docs/concurrency.md`
- `docs/debugging_generated_go.md`
- `docs/stability.md`
- `docs/compatibility.md`
- `docs/contributing.md`
- `docs/stdlib_style.md`
- `docs/testing_services.md`
- `docs/readiness.md`
- `PHASE8_NOTES.md`
- `PHASE9_NOTES.md`

## Compatibility / roadmap

Kiro is still experimental.

- Stable enough today: parser/formatter workflow, project/module boundaries, inspect-go workflow, starter templates.
- Likely to change soon: semantic/type system implementation details, executable backend code generation, stdlib runtime APIs.
- Near-term roadmap: complete Phase 9 hardening milestones (stdlib consistency, sema/codegen invariants, snapshots, tiny-project acceptance, and contributor workflow polish).
