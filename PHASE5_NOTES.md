# PHASE5 Notes

## What changed

This repository's Phase 5 update focuses on frontend and project ergonomics that are already represented in the current codebase:

- Added lexer support for:
  - `group` keyword reservation.
  - `///` doc comments as first-class tokens.
- Added parser support for doc comments on top-level declarations (`const`, `type`, `fn`).
- Added parser diagnostics for invalid doc comment placement (for example before `import`) and unattached doc comments.
- Extended AST declaration nodes to preserve attached doc comment lines.
- Updated formatter to emit preserved doc comments in canonical `/// ...` form above declarations.
- Added/updated tests for lexer, parser, and formatter around Phase 5 additions.
- Added requested Phase 5 example programs:
  - `examples/group_parallel`
  - `examples/env_config`
  - `examples/http_query`
  - `examples/http_headers`
  - `examples/test_demo`
  - `examples/parse_numbers`
  - `examples/doc_comments`
- Updated `README.md` to reflect current Phase 5 frontend-oriented status.

## Known limitations in this slice

This repository currently remains a frontend-focused milestone and does not yet include full runtime/backend implementation for many service features listed in the Phase 5 prompt:

- `group { ... }` is keyword-reserved and demonstrated in examples, but full structured-concurrency semantic enforcement and lowering are pending.
- `spawn`/`await` runtime behavior, task tracking, and diagnostics for un-awaited tasks are pending.
- Env/parse/http/log/test stdlib helper expansions are represented as target APIs in examples, but concrete runtime modules are pending in this slice.
- Manifest/module-resolution changes were not introduced in this phase because current repo scope does not yet require a manifest implementation.
- Optional-specific helpers and richer service diagnostics remain future work.

## Why `?` remains preferred error flow primitive

A new `try { ... }` block was intentionally not added in this phase to keep grammar complexity low while frontend/runtime layers are still converging. The existing postfix `?` remains the preferred error propagation primitive for now.

## Recommended Phase 6 scope

1. Implement structured concurrency semantics for `group`:
   - track spawned tasks within lexical group scopes
   - warn/error for obviously un-awaited tasks
   - maintain readable Go lowering
2. Implement stdlib/runtime helpers for:
   - `env.get`, `env.get_or`, `env.must`, numeric env parsing
   - `parse.i32`, `parse.i64`, `parse.bool`
   - `http.query`, `http.header`, `http.with_header`
   - minimal `log` and `test` helper APIs
3. Expand diagnostics with source-span snippets for common service mistakes.
4. Add end-to-end example execution checks once `build/run/test` move beyond placeholders.
