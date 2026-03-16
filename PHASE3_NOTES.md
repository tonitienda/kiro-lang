# PHASE3_NOTES

## What changed

Phase 3 has started with Milestone 1 parser/formatter groundwork:

- Added value-receiver method declaration syntax to the AST/parser/formatter.
- Added tests that parse and format method declarations.
- Expanded reserved keyword recognition for upcoming loop/defer work (`while`, `break`, `continue`, `defer`).
- Added `examples/methods/main.ki` showing method declaration and call shape.
- Updated README and syntax docs to match the implemented frontend surface.

## Known limitations

Current repository state still focuses on frontend syntax handling. The following Phase 3 items are not implemented yet:

- code generation / runtime semantics for methods
- package-qualified member resolution behavior
- map literals/types/lookup
- `while`, `break`, `continue` parsing and lowering
- enhanced `when` constructor matching for `R[T,E]`
- `defer` statement parsing/lowering/validation
- stdlib expansions (`cli`, `env`, `log`, `time`, improved `http`/`json`)
- `kiro test` feature work
- diagnostics improvements listed in the Phase 3 plan
- full new example suite listed in the Phase 3 prompt

## Recommended Phase 4 scope

After Phase 3 milestones are fully delivered, prioritize:

1. Stabilize parser + formatter around control flow and map syntax.
2. Connect frontend constructs to semantic analysis and Go lowering with end-to-end tests.
3. Implement practical stdlib/runtime glue for CLI + HTTP + JSON demos.
4. Upgrade diagnostics to include clearer context and actionable messages.
