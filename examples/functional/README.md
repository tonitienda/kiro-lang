# Explicit functional basket pricing

The pricing module builds a typed `Transform[Item,Line]` from a filter and a specialized
pricing function. `list.transduce` executes it in one processing loop, accumulating a total
without either intermediate list. `total_eager` retains the separate eager filter/map/reduce
implementation for comparison. Both use pure callbacks.

```sh
go run ./cmd/kiro fmt examples/functional
go run ./cmd/kiro check examples/functional
go run ./cmd/kiro run examples/functional
go run ./cmd/kiro test examples/functional
```

The program prints `3700` cents, selecting the declared zero-discount default explicitly.
Four tests cover numerical eager composition, typed function factories, capture snapshots,
nested partials/thunks, discounted and empty baskets, eager/fused total equivalence, and
reusing the same transform with `list.collect`. The discounted basket totals 3500 cents.

See [the functional contract](../../docs/functional-core.md) for execution/error order and
[allocation measurements](../../docs/research/transducer_measurements.md) for runtime limits.
This example exercises implemented semantics; it is not an agent benchmark.
