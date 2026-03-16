# PHASE2 Notes

## What was implemented

- Added parser support for block-bodied functions:
  - `fn x(...) -> T = ...`
  - `fn x(...) -> T { ... }`
- Added `FuncDecl.BlockBody` flag in AST so formatter and future phases can distinguish body forms.
- Extended parser import handling to support deterministic local paths like `app/router`.
- Updated formatter to print block-bodied functions canonically.
- Added formatter normalization for tokenized operators (`==`, `=>`, `->`) that are lexed as separate symbols.
- Added parser/formatter tests for the new syntax.
- Updated docs and examples for current Phase 2 status.

## Known limitations

- Function bodies are still stored as normalized text, not structured statement/expression AST nodes.
- No semantic analysis, codegen, stdlib runtime, or CLI build/run/test execution yet.
- Block internals (`return`, assignment, loops, `when`) are accepted as text, but not type-checked.
- Import support is lexical/path parsing only; module resolution is not implemented.

## Recommended Phase 3 next steps

1. Introduce statement/expression AST nodes for function bodies.
2. Add semantic analysis for assignment mutability, returns, and `if/when` typing.
3. Implement minimal Go backend for executable examples.
4. Wire `kiro build/run/test` to parser + checker + backend pipeline.
5. Expand stdlib (`fs`, `http`, `json`) once codegen/type checking are stable.
