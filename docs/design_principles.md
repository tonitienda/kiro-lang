# Design principles

Kiro is being redesigned around a strict machine-friendly core.

## 1. One obvious way

Every important construct should have one obvious syntax and one obvious documented style.

Current canonical choices:

- block-only function bodies
- explicit `return`
- effect markers after return types
- `R[T,E]` for failure
- `?T` for optionality
- `group { ... }` for structured concurrency scope

## 2. Concepts stay orthogonal

Kiro sharply separates:

- impurity
- failure
- absence
- concurrency

If two ideas can be modeled separately, Kiro should not blur them for convenience.

## 3. Explicitness beats convenience

Prefer:

- explicit effects
- explicit returns
- explicit result propagation
- explicit optional types
- explicit concurrency boundaries

Avoid hidden inference that makes generated code harder to audit.

## 4. Canonical formatting is mandatory

`kiro fmt` is not cosmetic. It defines the canonical printed form.

## 5. Diagnostics are part of language design

A language optimized for LLM maintenance must make common repair steps local and obvious.

## 6. Generated Go remains inspectable

Lowering stays visible through `kiro inspect go`.
