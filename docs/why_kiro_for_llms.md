# Why Kiro for LLMs

Kiro is being shaped around a simple idea: **make the language easy to generate correctly, easy to format canonically, and easy to repair from diagnostics**.

## Core properties

### 1. Small surface area

Kiro intentionally stays narrow:

- modules
- structs
- functions
- results and optionals
- explicit effects
- structured concurrency
- HTTP/JSON-oriented server-side workflows

### 2. One obvious syntax per construct

The current optimization pass standardizes on:

- **block-only function bodies**
- explicit `return` for value flow
- explicit effect markers after return types
- `group` / `spawn` / `await` for concurrency
- one canonical formatter output

### 3. Orthogonal semantics

Kiro keeps these concepts separate:

- **effects** = operational impurity
- **`R[T,E]`** = fallibility
- **`?`** = result propagation
- **`?T`** = optionality
- **`nil`** = absence for optionals

That separation matters for both code generation and diagnostics.

### 4. Canonical formatting

A formatter is part of the language design, not an afterthought.

Canonical formatting helps LLMs because:

- training examples converge on one printed shape
- repair loops can suggest exact rewrites
- diffs stay small and semantic

### 5. Compiler-guided repair

The repository now prefers diagnostics that say:

- what is wrong
- which construct is wrong
- what to add, remove, or replace

Examples:

- missing effect declarations
- old expression-bodied functions
- invalid `?` on non-`R[T,E]` values
- `await` used on a non-task

### 6. Stable project conventions

Kiro is not only a language; it also suggests one obvious project shape:

- `main.ki` for entrypoints
- `app/` for handlers
- `internal/config/` for environment-derived config
- `test/` for direct handler tests

### 7. Inspectable lowering

Kiro lowers to Go and keeps that path inspectable with `kiro inspect go`.

That makes debugging more practical for both humans and tool-assisted workflows.

## What Kiro deliberately avoids

- operator overloading
- effect inference
- syntax aliases that compete with the canonical style
- hidden concurrency or async magic
- conflating JSON/parsing with effects

## Repository support for this design

The design is reinforced by:

- compatibility fixtures
- formatter tests and idempotence checks
- template verification
- example checks
- editor/LSP hover signatures sourced from the same compiler surface
