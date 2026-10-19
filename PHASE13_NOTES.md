# Phase 13: explicit functional core

The experiment keeps agent implementation, review, audit and deterministic checks, without
a human code-review gate. Phase 12's keyed calls and delimited cases remain the base.

Implemented:

- `Fn[(key:Type)->Return!effect]` function contracts, including nested types; labels carry
  meaning across higher-order boundaries, and actual effects must fit the declared bound.
- `partial f(key:value)` with explicit subset binding, pure eager captures, immutable data
  snapshots, preserved latent effects, nested specialization and fully bound thunks.
- Scalar literal declaration defaults, selected only by an explicit `key:default` marker.
  Ordinary calls still supply every key. Defaults are not assumed by typed function APIs.
- Eager `list.map`, `list.filter`, and `list.fold`, with pure keyed callbacks and checks
  for invalid empty-list callbacks/reducer contracts.
- Executable-expression checking for function values, known value-type mismatches, aliases,
  default use, immutable binding assignment and basic optional narrowing. Runtime argument
  and return contracts supplement it. Legacy unsupported bodies are not fully type checked.
- Strict checked i32 arithmetic, relational/equality operators and negative integer literals.
  Fixed spawn precedence so the call, rather than just a function reference, is spawned.
- Qualified record literals, supporting collection examples across module boundaries.
- A two-module functional basket example with three tests and synchronized compact guidance.

See `docs/functional-core.md` for precise semantics and remaining limitations. The guide
snapshot identifies `phase13-functional-core` and is copied automatically by `kiro new`.
No automatic currying, laziness, lambdas, pipelines, ownership types, tagged variants/unit,
full schema validation, exhaustive matching or complete result/concurrency enforcement is
claimed. `inspect go` remains structural, not typed lowering of executable business logic.

Validation includes negative checker tests, direct runtime tests (which deliberately skip
checking), native compilation/execution of the functional example, existing compatibility
fixtures, and the quote-service suite. Existing contract probes are observations; the wrong
return-type probe now fails at checking instead of silently running. The other known gaps
remain recorded, rather than being treated as fixed by the new features.
