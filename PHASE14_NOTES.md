# Phase 14: explicit ordered transducers

Agent implementation, audits and deterministic verification continue without a required
human source-review stage. This batch addresses intermediate collection storage and naming.

- The canonical terminal is `list.reduce(items:xs, initial:zero, reduce:r)`, with all arguments keyed
  in source. It is an ordered left fold with a required initial value, including for an
  empty list. `list.fold` is removed with a migration diagnostic; it is not a second alias.
- `list.mapping(transform:f)` and `list.filtering(predicate:p)` construct immutable,
  source-independent `Transform[Input,Output]` values. They snapshot their pure callbacks
  but do not invoke them. Function/transform boundaries retain declared type contracts.
- `list.compose(first:a, then:b)` builds a flattened plan in explicit first-then order.
  Stage connections are checked before processing. Plans can be returned, passed, aliased,
  reused, and captured by partial applications.
- `list.transduce(items:xs, through:plan, initial:zero, reduce:r)` executes each item's stages
  and reduction before advancing to the next item. Rejected items skip remaining stages.
  There are no intermediate data lists; plan storage depends on the stage count.
- `list.collect(items:xs, through:plan)` realizes just the final output list. Both terminals
  are eager. Existing `map`/`filter` calls still materialize their output lists.
- The first runtime failure aborts processing, with item/stage context. Explicit fusion
  changes failure order relative to multiple eager passes; it is not a hidden optimizer.
- Shared callback validation enforces pure named callbacks, boolean predicates and consistent
  reducer state even on empty input. Runtime boundaries still validate actual values.
- Removed redundant scanning of `List[Any]` values during wildcard boundary validation;
  ordinary typed list/function contracts remain checked.
- The functional basket example compares eager and fused totals and reuses typed plans.
  Compact guide snapshot: `phase14-transducers`.

Tests compare both implementations with an independent Go reference over varied input
lengths; cover capture isolation, type/label/effect failures, empty inputs and failure
ordering; and compile/execute the example. `BenchmarkCollectionPipelines` records timing,
bytes and allocations separately. See `docs/research/transducer_measurements.md` for
observations and limits: eliminating intermediate lists does not remove per-element
interpreter allocations or establish a runtime speed advantage.

Only stateless one-to-one mapping and filtering stages are implemented. There is no lazy
sequence, parallel reduction, implicit fusion, stateful take/partition stage, early stopping,
completion protocol or general stream/channel adapter. JSON, failed-result propagation,
record identity/schema and wider concurrency/control-flow gaps are unchanged.

Phase 15 supersedes result propagation, record schema and JSON limits described above;
see `PHASE15_NOTES.md`. Its added checks are not covered by these historical measurements.
