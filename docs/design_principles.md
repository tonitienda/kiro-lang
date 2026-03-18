# Kiro design principles

Kiro aims to stay small, explicit, and server-focused.

## Principles

1. **Small language first**
   - Prefer a compact syntax and avoid feature sprawl.
2. **Explicit semantics**
   - Keep behavior understandable from source; avoid hidden magic.
3. **Canonical formatting**
   - `kiro fmt` is deterministic and part of normal workflow.
4. **Backend pragmatism**
   - Go backend remains central for compilation and execution strategy.
5. **Server-side focus**
   - Prioritize service/CLI/file-processing workloads over browser targets.
6. **Result/optional discipline**
   - `R[T,E]` and `?` are the primary error flow; `?T` models absence clearly.
7. **Explicit operational effects**
   - Environment, filesystem, I/O, logging, networking, process, and time behavior should be visible in function signatures.
8. **Tiny structured concurrency**
   - `group`/`spawn`/`await` should stay readable and testable.
9. **Stdlib restraint**
   - Keep APIs small, consistent, and intentionally named.
10. **Compatibility discipline**
    - Stable-core behavior is guarded by fixtures and CI.
11. **Generated Go as debugging surface**
    - Inspection output is a first-class tool for understanding compiler behavior.
