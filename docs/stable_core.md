# Stable experimental core (Phase 10)

This page defines the **stable experimental core** for early Kiro adopters.

## What is stable-enough now

The following areas are source-compatibility sensitive and treated as stable experimental core:

- module layout and imports (`mod`, `import`, project entry `main.ki`)
- structs, constants, and value-receiver methods
- block functions and expression functions
- explicit function effect declarations (`!env`, `!fs`, `!io`, `!log`, `!net`, `!panic`, `!proc`, `!time`)
- local bindings and reassignment
- control flow: `if`, `when`, `for in`, `while`, `break`, `continue`
- collections: lists and maps
- result and optional patterns: `R[T,E]`, `?`, `?T`, `nil`
- string interpolation
- doc comments attached to declarations
- `defer`
- tiny structured concurrency: `spawn`, `await`, `group`
- canonical CLI workflow: `fmt`, `check`, `inspect go`, `new`, `compat`
- core server stdlib modules used by templates/examples (`http`, `json`, `env`, `log`, `test`, `ctx`, `parse`, `fs`)

## Experimental or likely to change

These areas are intentionally still more flexible:

- exact diagnostics wording (signal should stay stable)
- full semantic edge-case coverage for advanced type interactions
- generated Go internals and helper/runtime file details
- placeholder command behavior (`kiro build`, `kiro run`, `kiro test`) in this repository slice
- less-common stdlib helpers not used by stable examples/templates

## Intentionally out of scope

Kiro Phase 10 does **not** target:

- non-Go backend targets (JS/WASM/native codegen)
- self-hosting compiler work
- ownership/borrow systems
- macros
- package registry/distribution system
- trait/typeclass style abstractions
- large framework surface expansion

## How to decide if a change belongs in the core

A change belongs in stable core only when it is:

1. needed for README/template/example coherence,
2. covered by compatibility/tests,
3. documented in user-facing docs, and
4. expected to remain useful across phases.
