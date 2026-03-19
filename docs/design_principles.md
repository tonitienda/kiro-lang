# Design principles

Kiro is being refined around a deliberately machine-friendly core.

## 1. Small surface area

Prefer a smaller language with fewer competing forms.

## 2. One obvious syntax per construct

- block-only function bodies
- explicit effect placement after return types
- canonical formatting

## 3. Orthogonal semantics

Keep these distinct:

- effects
- results
- optionals
- concurrency

## 4. Explicitness over inference

Kiro favors visible signatures, visible returns, and visible task lifecycles.

## 5. Repairable diagnostics

Compiler feedback should help users make a concrete edit, not just describe a failure.

## 6. Inspectable lowering

Generated Go remains part of the trust and debugging story.
