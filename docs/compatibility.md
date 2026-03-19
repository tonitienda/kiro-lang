# Compatibility fixtures

Kiro's compatibility corpus protects the **redesigned language core**, not historical accidents.

## Fixture classes

- `stable_core/` for canonical successful programs
- `diagnostics/` for stable repair-oriented failures
- `regression/` for semantic guarantees that must stay true

## Current focus

The corpus should verify at least:

- formatter idempotence
- project loading and import resolution
- effect diagnostics
- pure JSON/parse semantics
- generated-Go inspectability for canonical projects

## Command

```bash
go run ./cmd/kiro compat
```
