# Phase 2 Roadmap (Incremental)

This repository is implementing Phase 2 incrementally.

## Completed in this slice

- Block-bodied function parsing (`fn ... { ... }`) alongside existing expression-bodied form.
- Basic formatter support for block-bodied functions.
- Parser support for slash-separated import paths (`import app/router`).

## Next milestones

1. Statement-level parsing inside blocks (`return`, assignment, block `if`, `for in`).
2. Expression improvements (`when`, list literals, indexing, `len`).
3. Type/semantic diagnostics and Result (`R[T,E]`, `?`) flow.
4. Stdlib/module improvements and end-to-end examples.
