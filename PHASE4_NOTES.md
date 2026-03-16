# PHASE4_NOTES

## What changed

Phase 4 has started with a focused frontend and documentation slice:

- Added module-level `const` declarations to AST/parser/formatter.
- Added optional type reference parsing/formatting for `?T` in struct fields, params, and return types.
- Reserved `spawn` and `await` as keywords in the lexer for upcoming concurrency milestones.
- Added parser/lexer/formatter tests for the new `const` + optional-type syntax and interpolation string stability.
- Added new Phase 4 example programs:
  - `examples/spawn_await`
  - `examples/http_parallel`
  - `examples/http_context`
  - `examples/optional_fields`
  - `examples/interpolation`
  - `examples/constants`
- Updated README and syntax docs to reflect current implemented frontend surface.

## Known limitations

This repository is still in a frontend-centric state. The following Phase 4 items are not implemented yet:

- `spawn`/`await` parsing, type-checking, and Go lowering
- stdlib/runtime task helpers (`task.join`, etc.)
- improved `R[T,E]` constructor ergonomics and result diagnostics
- optional runtime semantics beyond parsing `?T`
- string interpolation expression parsing/lowering semantics (currently preserved as string content)
- `ctx`, `http`, `json`, `test` stdlib implementation work
- `kiro test` behavior beyond placeholder command wiring
- end-to-end Go code generation/build/run verification for new examples

## Recommended Phase 5 scope

1. Implement parser+AST+formatter support for `spawn` and `await`, including targeted diagnostics.
2. Add semantic and lowering support for package-level `const` and `?T` optional values.
3. Deliver a minimal Go backend runtime shim for spawned tasks and blocking `await`.
4. Expand stdlib modules (`ctx`, `http`, `json`, `test`) with small, stable APIs and e2e coverage.
5. Improve diagnostics for service-code workflows and document final Phase 4 syntax/limitations clearly.
