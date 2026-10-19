# Phase 12: agent-only core reassessment

- Agents implement, review, audit and create deterministic validation tools; human source
  review is outside this experiment's objective.
- `when (subject) { pattern => { ... } }` is the only supported case-header/block form.
  Old unbraced headers fail with a migration diagnostic. Parentheses remove subject/record
  literal ambiguity, and braces let execution continue after a case block.
- Updated examples, compatibility fixtures, service scaffolding and the bundled skill.
  The Phase 12 compact manifest identified `phase12-keyed-calls`; Phase 13 supersedes it. Regenerate downstream snapshots.
- The formatter preserves string contents while normalizing surrounding non-literal tokens.
- `docs/research/agent_core_reassessment.md` distinguishes implemented changes from proposed
  tagged optionals/unit, typed errors/exhaustiveness and structural agent tools.
- Required keyed value calls (`f(name: expression)`) are implemented for functions, methods,
  built-ins and late-bound function values. Reject positional/mixed/duplicate/unknown/missing
  arguments. Values evaluate once in source order and bind by name; formatting preserves order.
- Shared built-in parameter schemas drive checking/runtime binding and are regression-checked
  against the compact agent manifest. Parameter names now form part of the callable API.
- Checking validates signatures for directly resolved calls; unresolved higher-order calls
  receive runtime signature validation before argument evaluation. This is not full type checking.
- Fixed method argument binding to account correctly for the implicit receiver. Record
  construction uses keyed record literals; calling a type as a function is rejected at runtime.
- Layered verification deliberately varies inputs/goals and measures overlapping detections,
  false alarms and escaped defects; different agents are not assumed statistically independent.
- Phase 10's results/effects/optionals/concurrency concepts remain the design base. Existing
  body/type/JSON/result/concurrency enforcement gaps remain until explicitly implemented.

Phase 13 adds typed function values, partial application, explicit defaults and pure eager
collection transforms; see `PHASE13_NOTES.md` for the current enforcement boundary.
