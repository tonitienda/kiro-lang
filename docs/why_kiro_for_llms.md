# Why Kiro is optimized for LLMs

Kiro is being shaped as a language that is easier for models to **generate**, **review**, **repair**, and **maintain**.

## What Kiro optimizes for

### 1. Small language, high semantic density

Kiro intentionally keeps the core small:

- one declaration style
- one function body style
- one explicit effect system
- one result model
- one optional model
- one structured concurrency model

### 2. Orthogonal concepts

Kiro keeps these ideas separate:

- **effects** = operational impurity (`!fs`, `!net`, `!env`, ...)
- **`R[T,E]`** = fallibility
- **`?`** = result propagation
- **`?T`** = optionality
- **`nil`** = absence where the type allows it
- **`group` / `spawn` / `await`** = concurrency lifecycle

### 3. Canonical syntax

The formatter is part of the language definition.

A model should not have to choose among multiple equally-valid surface forms. Kiro therefore prefers:

- block-only functions
- explicit return statements
- explicit effect markers after return types
- canonical lexicographic effect ordering
- stable project layout

### 4. Compiler-guided repair

Diagnostics are designed to support a repair loop:

- identify the local cause
- state what was expected
- suggest the next edit

This matters especially for:

- missing effects
- invalid pseudo-effects like `!json`
- invalid imports
- misuse of `?`, `?T`, and `nil`
- malformed concurrency structure

### 5. Inspectable backend lowering

Kiro keeps generated Go visible so that both humans and models can debug what the compiler emitted.

## Compact downstream package

Downstream repositories should not have to ship the full `kiro-lang` repo into every prompt.

Use the compact package in `docs/llm/` instead:

- `docs/llm/KIRO_SKILL.md` for concise prompt-ready language guidance
- `docs/llm/kiro.json` for machine-readable conventions
- `docs/llm/examples/` for short canonical examples

This package is intentionally smaller than the full docs. It should stay in sync with the stable core and canonical stdlib/project guidance.

## What Kiro deliberately avoids

To stay machine-friendly, Kiro avoids or limits:

- multiple equivalent syntaxes for major constructs
- effect inference
- operator overloading
- magic convenience helpers with unclear semantics
- conflating failure with impurity
- ambient concurrency

## Design consequence

Kiro is intentionally more opinionated and narrower than a general-purpose language prototype. That tradeoff is deliberate: less choice and less ambiguity generally produce better generated code and more reliable repair.
