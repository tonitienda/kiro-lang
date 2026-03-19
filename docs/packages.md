# Packages and project boundaries

Kiro projects resolve imports through declared module names and predictable path mapping.

## Practical workflow

- `kiro check` validates the project graph.
- `kiro inspect go` preserves relative paths in generated inspection output under `.kiro-gen/src`.
- `kiro test` runs `test_*` functions through the standalone build path.
- `kiro build` produces a native executable for the project entrypoint.

## Current guidance

Use `kiro inspect go` when you want to reason about generated source layout, and use `kiro build/run/test` when you want the standalone toolchain workflow that downstream repositories will consume.
