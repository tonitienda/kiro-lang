# Explicit functional core

Phase 13 adds explicit function specialization without automatic currying. All ordinary
calls still supply every key. This is an experimental contract for agents implementing,
reviewing and auditing code without a human source-review gate; it is not a measured
improvement in model performance.

## Typed function values

```ki
fn apply(value:i32, transform:Fn[(value:i32)->i32])->i32 {
  return transform(value:value)
}
```

`Fn[(value:i32)->i32]` specifies parameter names, types and return type. Effects belong
inside the brackets: `Fn[(value:str)->nil!io]`. No effect suffix means pure. A pure
function may satisfy an effectful bound; the reverse is rejected. Parameter order may
vary, but names and types must agree. Passing and returning functions uses these contracts;
functions can also be aliased with `let`. Nested function types are supported.

Defaults are declaration metadata, not part of a `Fn` type. Code calling through a typed
function parameter must supply concrete values; it cannot assume a default exists.

## Explicit partial application

```ki
fn add(value:i32, amount:i32 = 1)->i32 {
  return value + amount
}

fn offset(amount:i32)->Fn[(value:i32)->i32] {
  return partial add(amount:amount)
}

fn example()->i32 {
  let increase = offset(amount:3)
  return increase(value:4)
}
```

`partial f(key:value)` creates a function over the unbound parameters, preserving their
names, types, defaults and the target's latent effects. At least one parameter must be
bound. Binding all parameters creates a zero-argument function, invoked with `f()`.
Partial functions may be specialized again. Methods and built-in functions are supported,
including retaining the type argument in `partial json.decode[Input](text:body)`.

The callee expression and capture expressions must be pure. They evaluate once at
creation, in source order. Captured records, lists and nested partials are copied; bound
method receivers are copied too. Later rebinding cannot change a capture. Each invocation
gets a fresh copy of captured data. Futures and unsupported runtime handles cannot be
captured. Calling the target happens only when the resulting function is invoked; its
declared effects must then be permitted by the caller.

`let` bindings reject reassignment; use the existing `mut name = value` form for local
rebinding. Parameters are immutable. These are binding and capture guarantees, not a
complete ownership system or global proof of immutability.

## Visible defaults

```ki
fn example()->i32 {
  return add(value:4, amount:default)
}
```

The key stays visible. Omitting `amount` remains an error. Defaults must be scalar literals:
signed integers in range, strings without interpolation, booleans, or `nil` for a compatible
type. They are validated even if never selected. Defaults cannot call functions, read
configuration, refer to another parameter, or allocate mutable collections. The bare
`default` marker is valid only as an entire keyed argument, including in `partial`.

## Eager collection transforms

| Helper | Required keys | Pure callback contract |
| --- | --- | --- |
| `list.map` | `items`, `transform` | `value:T` returns an output value |
| `list.filter` | `items`, `predicate` | `value:T` returns `bool` |
| `list.reduce` | `items`, `initial`, `reduce` | `state:S`, `value:T` returns `S` |

Transforms run eagerly in list order. Reduce is an ordered left fold, with an explicit
initial value. An empty reduction returns that initial value. Callback labels, purity, filter result type and
reducer state/return compatibility are checked even for empty lists. Typed item and state
values are checked at invocation. A mapped `R[T,E]` remains a result value; list helpers
neither unwrap results nor silently discard failures.

The runnable [basket pricing example](../examples/functional/README.md) composes filtering,
a specialized price function and a total reducer across modules. Run:

```sh
go run ./cmd/kiro run examples/functional
go run ./cmd/kiro test examples/functional
```

## Explicit fused transducers

```ki
fn plan()->Transform[i32,i32] {
  return list.compose(first:list.filtering(predicate:positive), then:list.mapping(transform:increment))
}

fn total(items:List[i32])->i32 {
  return list.transduce(items:items, through:plan(), initial:0, reduce:sum)
}
```

Here `positive(value:i32)->bool`, `increment(value:i32)->i32`, and
`sum(state:i32,value:i32)->i32` are ordinary pure block functions. The complete runnable
forms are in the basket example and `compat/stable_core/transducers/main.ki`.

| Helper | Required keys | Result |
| --- | --- | --- |
| `list.mapping` | `transform` | `Transform[T,U]` from a pure `value:T -> U` callback |
| `list.filtering` | `predicate` | `Transform[T,T]` from a pure `value:T -> bool` callback |
| `list.compose` | `first`, `then` | Typed first-then composition |
| `list.transduce` | `items`, `through`, `initial`, `reduce` | Final accumulator |
| `list.collect` | `items`, `through` | Only the final output list |

Plans contain no source list. Construction snapshots callbacks, including partial captures
and method receivers, without executing them. Plans can be reused over different input
lists, returned from functions, passed through typed boundaries and captured in partials.
The output of one stage must fit the next stage's input. Reducer input and state contracts
are checked before iteration, including on empty input. A callback accepting `Any` can
consume concrete values; an `Any` output cannot promise a concrete receiving type.

Both terminals execute immediately, in input order. For each item, stages execute in
first-then order, followed by reduction or final collection. A failed predicate skips later
stages and reduction for that item. The first runtime failure aborts; errors identify the
zero-based item and stage (or reducer). Result values are ordinary outputs, not implicitly
unwrapped. An empty transduction returns its required initial value; empty collect returns
an empty list.

This is an explicit execution model. Separate `filter`/`map` calls still finish one full
pass before the next starts. As a result, they can report a different first error from a
transducer pipeline even with pure callbacks. Kiro does not silently fuse existing calls.

Implementation: immutable flattened stage descriptors drive a single processing loop.
Transduce holds one current element and accumulator, with no intermediate data lists;
collect allocates only its final result list. Descriptor storage grows with stage count.
Input data, the accumulator, callback-produced values, interpreter frames/argument storage,
and boundary validation still have costs. There is no claim of zero allocation or faster
execution; see [the measurements](research/transducer_measurements.md).

`list.reduce` replaces `list.fold`; there is one canonical name. Reduce remains sequential,
with no associativity or parallel-execution requirement. Only stateless mapping/filtering
stages over finite lists are supported in this batch. There is no lazy collection, stateful
stage, early-termination/completion protocol, stream adapter or general pipeline operator.

## Enforcement and limits

Checking resolves supported executable expressions, aliases and function contracts. It
rejects known argument/return type mismatches and narrows an optional in a wildcard arm
following an explicit `nil` case. Unknown expression types are deferred to runtime checks.
Legacy frontend-only control constructs are still outside the executable checker; a body
using the new functional constructs must parse in the executable subset.

Runtime calls validate parameter and return contracts, including nested lists, results and
function signatures. Records currently check nominal names, not full field schemas or
module-distinct type identity. JSON validation, compositional failed-result propagation,
exhaustive typed variants, and complete concurrency/effect enforcement remain unfinished.
Effects here are declared operational contracts, not a security sandbox. Functions reached
through host callbacks and unsupported syntax still need broader analysis. `kiro inspect
go` remains structural output, while runnable builds use the Go-hosted interpreter.

Numeric arithmetic checks i32 bounds and division by zero, with comparisons producing
booleans. String `+` concatenates strings; numeric-looking strings are not coerced to
integers. Negative integer literals are supported. Arithmetic errors currently abort
execution rather than returning a typed result.

No implicit currying, anonymous functions, lazy collections, pipeline operator syntax, tagged
optionals/unit or new variant syntax are introduced in this pass. Those require separate
semantic work and controlled comparison experiments.
