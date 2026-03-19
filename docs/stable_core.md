# Stable core

The stable core is intentionally narrow.

## Included

- deterministic lexer/parser/formatter behavior for the documented syntax
- block-only function declarations
- explicit effects in signatures
- explicit result/optional separation
- structured concurrency with `group`, `spawn`, and `await`
- predictable project/module resolution
- canonical CLI workflow: `fmt`, `check`, `inspect go`, `build`, `run`, `test`, `new`, `compat`, `lsp`
- generated-Go visibility as part of the workflow

## Protected by the repository

- Go tests across parser/formatter/project/build/LSP layers
- compatibility fixtures
- examples and template verification
- formatter idempotence through compatibility checks

## Intentionally not promised

- a frozen generated-Go API
- broad runtime coverage for every experimental example
- effect polymorphism or large-scale type inference
