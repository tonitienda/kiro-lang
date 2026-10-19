# Explicit functional basket pricing

The pricing module filters out zero-quantity items, maps an explicitly specialized price
function over the remaining items, and folds priced lines into a total. A function factory
returns `Fn[(value:Item)->Line]`, preserving roles across the module boundary.

```sh
go run ./cmd/kiro fmt examples/functional
go run ./cmd/kiro check examples/functional
go run ./cmd/kiro run examples/functional
go run ./cmd/kiro test examples/functional
```

The program prints `3700` cents, selecting the declared zero-discount default explicitly.
The three tests cover a numerical pipeline, typed function arguments/factories, immutable
captures after local rebinding, nested partials/thunks, and discounted/empty baskets.
The discounted basket totals 3500 cents.

See [the functional contract](../../docs/functional-core.md) for syntax and enforcement
limits. This example exercises implemented semantics; it is not an agent benchmark.
