# Phase 17: agent/compiler feedback and discovery

Agent implementation, audit and deterministic verification continue without a human
source-review gate. This batch makes the language easier to inspect and repair with tools.

- Added `kiro.tools.v1` JSON envelopes to check, lint, describe, symbols and fmt --check.
  Validation failures exit nonzero with one stdout report and no duplicate stderr prose.
- Added shared structured diagnostics for native syntax, type, keyed-argument, effect,
  missing-return and missing-variant checks, with broad fallback codes for legacy errors.
  Source ranges distinguish exact tokens, enclosing declarations and legacy positions.
- Added conservative intraprocedural return-path checking. Loops may be empty and ? may
  succeed; neither alone proves an exit. Nil/optional/Any fallthrough remains admitted.
- Added language/builtin discovery from the compact guide and executable signature registry,
  plus syntax-only project symbol queries with module-qualified IDs and source hashes.
- Added read-only formatter checks and lint (semantic validation then canonical formatting).
  Formatting preflights all files, preserves permissions, and uses deterministic traversal.
  Hello/service scaffolds are emitted canonically and pass lint immediately.
- LSP uses the semantic checker with in-memory unsaved overlays, shared codes/data, source
  routing/clearing and UTF-16 ranges. Completion includes real builtin/constructor contracts
  and stops suggesting unsupported control-flow constructs.
- Added a reusable maintenance-task preparer/evaluator for stock errors, request schema
  changes and coupon validation. It checks fixed acceptance behavior in an isolated copy;
  agent metrics and comparative runs remain unmeasured.
- Updated variant evolution probes to consume structured fields rather than prose.
- Compact guide version: phase17-agent-tools. See docs/agent-tools.md for commands, wire
  contracts, source coordinate rules, experimental setup and precise limitations.

Validation covers source/path contracts, JSON process behavior, discovery during broken
semantics, source-hash stability, formatter no-write/preflight behavior, native examples,
workspace overlays/diagnostic clearing, Unicode ranges, and deterministic maintenance probes.
No full call graph, inferred-local query, rename/structural edit protocol, complete static
type system, native typed Go lowering or concurrency/effect sandbox is claimed.

Phase 18 adds explicitly invoked analysis, coverage and mutation tools. They do not run in
normal compiler/linter/LSP workflows; see `PHASE18_NOTES.md` and `docs/analysis-tools.md`.
