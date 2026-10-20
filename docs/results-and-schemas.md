# Compositional results and validated data

Phase 15 makes existing result and record declarations constrain execution. It keeps
agent implementation, auditing and deterministic verification without a required human
source-review step. No agent-performance advantage is claimed.

## Function-local propagation

```ki
fn leaf()->R[i32,str] {
  return Err(error:"unavailable")
}

fn parent()->R[str,str] {
  let number = leaf()?
  return Ok(value:"${number}")
}

fn test_propagation()->nil {
  test.eq(actual:parent(), expected:Err(error:"unavailable"))
  return nil
}
```

`?` requires an enclosing `R[T,E]` return type, an actual result operand and a compatible
error type. On `Ok`, it yields the payload. On `Err`, it skips the remainder of that
function and returns the same error payload in its declared result channel. The success
types may differ: the example propagates an `R[i32,str]` failure as `R[str,str]`.

This includes errors originating in arguments, nested groups/cases and interpolation.
The propagation signal is consumed at the user-function boundary; it is not a general
catch-all. Type errors, invalid operators, arithmetic errors and failed assertions remain
runtime failures, rather than being converted into domain `Err` values.

`Ok(nil)` and `Err(nil)` have distinct explicit runtime tags. Typed validation selects the
active payload even when it is nil. Record fields, lists, captures and JSON round trips
preserve those tags. Assertions compare typed structural values; integer `1` does not
equal string `"1"`.

Partials and typed function values preserve function boundaries. Collections retain
callback result values: a mapper returning `R[T,E]` produces results, not implicitly
unwrapped values. A result-valued reducer can explicitly propagate from its state/input
and return an `Err` accumulator. This does not stop iteration automatically; early stopping
is a separate, unimplemented protocol.

For spawned work, call a result-returning function, await its result, then apply `?` in
the awaiting function. Inline propagation inside a `spawn` expression is rejected so a
control signal cannot outlive its enclosing function. Broader concurrency typing/lifecycle
semantics remain incomplete.

## Process, test and HTTP boundaries

A result-returning `main` unwraps one outer `Ok` and follows the existing integer exit-code
or nil-success convention. An unhandled `Err`, including `Err(nil)`, prints a diagnostic
and exits nonzero. A `test_*` function may return nil or a result; returning `Err` fails
the test. Assertions and runtime errors also fail tests.

Functions using `?` must migrate, for example:

```ki
fn main()->R[i32,str] !io {
  let number = parse.i32(text:"42")?
  println(value:number)
  return Ok(value:0)
}
```

The HTTP adapter accepts ordinary, partial and typed handler values. An outer `Err`
becomes HTTP 500; `Ok(http.Resp)` becomes a response. Domain-specific 400/422 handling is
still an application decision. Matching still compares whole values; general payload
binding, typed variants and exhaustiveness are not implemented in this batch.

## Closed record contracts

Source constructors require every declared field exactly once, including explicit nil
for an absent optional. Unknown and duplicate fields fail before any field expression
executes. Known type mismatches fail checking, and runtime boundaries validate actual
nested fields. Data and callable/result values remain native inside records; construction
no longer flattens them into display dictionaries.

Record values retain their declaring module, and qualified types are carried through
callable and JSON schema boundaries. A plain JSON dictionary cannot impersonate a record
by supplying an `_type` key. Ambiguous unqualified record names require a module qualifier.
Recursive schema declarations are allowed, with actual values checked recursively.

## Strict typed JSON

`json.decode[T](text:body)` requires one explicit schema type and returns `R[T,str]`.
Schema types include i32, str, bool, nil, optionals, lists, records and results. Unsupported
callable/transform/context schemas and unknown record schemas are program errors.
Malformed or nonconforming data returns `Err(str)` with a field/index path.

The decoder:

- Requires all nonoptional record fields and rejects unknown fields.
- Rejects duplicate object keys at every nesting level and trailing JSON values.
- Checks field/list-element types recursively and does not coerce strings or booleans.
- Fills omitted optional JSON fields with nil; explicit JSON null is valid only where the
  schema permits it. Source constructors still require explicit optional keys.
- Decodes i32 directly from integer number text, preserving range checks without float64
  rounding. Fractional/exponent spellings, including `1.0` and `1e0`, are rejected for i32.
- Requires UTF-8 text and limits input nesting to 128 levels. `Any` deliberately retains
  untyped JSON data, including exact `json.Number` values; it is not a schema guarantee.

`json.encode` emits only record data fields, without internal `_type` or module metadata.
Results encode as exactly one `{"Ok":payload}` or `{"Err":payload}` key; decoding an R schema
requires exactly that shape. The tag remains present for null payloads. Encoding runtime
functions/transforms/handles returns an `Err` rather than exposing their implementation.

String literals now support standard Go quoted-string escapes, including escaped quotes,
backslashes, newline and Unicode. Lexing/formatting preserve the source spelling; evaluation
decodes it once. Use `\\` for a literal backslash. Interpolation processes only original
source placeholders once: inserted data containing `${...}` is not executed again.

## Evidence and remaining work

The [validated aggregation example](../examples/validated_results/README.md) combines typed
JSON, optional fields, partial decoders, typed higher-order functions and result-valued
transduction across three modules. Tests also cover process/test/HTTP boundaries, explicit
nil tags, nested schemas, duplicate/trailing JSON, integer limits, module collisions,
escaped literals and interpolation data. A fuzz property validates successful decoded
records and checks encode/decode round trips.

The checker remains incremental; legacy unsupported bodies and unresolved expressions
are not a complete type system. Direct typed Go lowering, general error/variant matching,
exhaustive handling and complete concurrency semantics remain separate work. Arithmetic
errors still abort rather than becoming typed domain results. These semantics changes
have not been measured as an advantage in model generation, repair or auditing.
