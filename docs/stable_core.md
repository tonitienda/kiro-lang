# Stable core

The stable core is the intentionally small language slice that Kiro currently treats as canonical.

## Included in the stable core

- deterministic lexer, parser, and formatter behavior
- module/import loading for the documented project layout
- block-only function declarations
- explicit operational effects in function signatures
- `R[T,E]`, `?`, `?T`, and `nil` as separate concepts
- structured concurrency with `group`, `spawn`, and `await`
- generated-Go inspection with `kiro inspect go`
- compatibility fixtures for formatter, checking, diagnostics, and inspectability

## Stable-core style rules

- use one function style: `fn name(params) -> type !effects { ... }`
- prefer thin impure shells around pure inner logic
- keep handlers on `fn handler(req:http.Req) -> R[http.Resp,str]`
- keep config loading in dedicated `internal/config` modules
- keep tests in explicit `test_*` functions

## Not promised yet

- a frozen generated-Go API
- effect polymorphism
- broad inference-heavy typing
- multiple competing project styles
