# Typed variants, payload matching and explicit recovery

Phase 16 adds closed sum types to the agent-only language experiment. A declaration
contains record fields or `variant` declarations, never both:

```ki
type LookupError {
  variant Missing { key:str }
  variant Offline {}
}

fn explain(error:LookupError)->str {
  return when (error) {
    LookupError.Missing(key:let key) => "missing: ${key}"
    LookupError.Offline() => "offline"
  }
}
```

Construct values with keyed calls: `LookupError.Missing(key:"sku")` or
`LookupError.Offline()`. Constructors are pure typed callable values: they can be passed
through `Fn` contracts and specialized with `partial`. Every payload field is required,
including explicit nil optionals. Keys are checked before argument effects; arguments
are evaluated once in source order. Constructors and values retain their declaring module.
Two modules' identically shaped variants are distinct types.

Use `let name` inside a pattern to bind a payload; `_` ignores it. Keys use the same exact
names as construction. Bindings are immutable and scoped to their arm. Literals constrain
a payload instead of binding it. Pattern calls only destructure: they never invoke arbitrary
functions or execute interpolation. String patterns are literal text. Nested constructor
patterns are supported, but do not establish full coverage of their outer constructor.
Use a nested `when` to make payload decisions after a total outer binding.

## Coverage rules

For a known declared variant type, every variant must have a total arm: its payloads must
all be bindings or `_`. A payload literal such as `Missing(key:"only-this")` covers only
that value. Missing variants produce a diagnostic naming the subject type and missing tags.
An arm after a total arm for the same tag is rejected as unreachable. Declared variants do
not permit a wildcard catch-all: new variants must invalidate affected matches visibly.

For a known `R[T,E]` subject, both `Ok(value:...)` and `Err(error:...)` must be covered;
a final wildcard is also allowed for compatibility. `Ok(nil)` and `Err(nil)` are distinct.
Nonconstructor matches on strings, integers, booleans and optionals retain value dispatch
and optional narrowing. This is constructor coverage, not general constraint solving or
an exhaustiveness proof for arbitrary primitive/nested payload domains.

Expression arms must have the same inferred type. Block arms support explicit returns and
local propagation. Bindings do not escape the arm; writes to existing mutable outer bindings
remain visible. A declaration shadowing an outer name remains local. Unknown identifiers
are rejected. Unknown *expression types* can still reach runtime contract checks: this is
not a complete static type system or a proof that every function path returns a value.

## Explicit error conversion

`?` propagates only a compatible error channel. There is no implicit string conversion,
subtyping between unrelated error types, inferred union or automatic wrapping. Use ordinary
typed functions and exhaustive matches at the boundary:

```ki
fn amount()->R[i32,PublicError] {
  when (catalog.price(sku:"book")) {
    Ok(value:let number) => {return Ok(value:number)}
    Err(error:let failure) => {return Err(error:from_catalog(error:failure))}
  }
}
```

`from_catalog(error:catalog.Error)->PublicError` explicitly maps each source variant to a
public variant. Its new-variant diagnostic is the same as any other match. The quote service
uses this for catalog/shipping → pricing conversions and then maps all pricing failures to
HTTP 422. Malformed/nonconforming request JSON becomes HTTP 400. Unexpected serialization
errors remain unhandled Err and reach the HTTP adapter's 500 boundary.

## JSON and captures

Typed variants are JSON schema types. Their wire shape is exactly one tag with an object
payload: `{"Missing":{"key":"sku"}}` or `{"Offline":{}}`. Unknown/multiple tags, missing
required fields, unknown/duplicate keys and wrong nested payload types return Err. JSON
optional omissions become nil, while source constructors require all fields. Recursive
schemas and nested record/result/list payloads follow the existing JSON rules. Wire output
contains no module/type metadata. Runtime values and partial captures preserve module
identity; capture copies prevent later local rebinding from changing the captured payload.

## Executable list loops and honest checking

```ki
mut total = 0
for item in [1,2,3] {
  total = total + item
}
```

`for name in listExpression { ... }` evaluates its finite list once and visits items in
order. Each immutable iteration binding and body-local declaration has its own scope.
Outer mutable updates are preserved; nested loops, explicit return and function-local `?`
work normally. Empty lists execute no body. The checker requires a known list type.
Parenthesize complex iterable expressions where needed to delimit them from the block.

All function bodies must now parse in the executable grammar during `kiro check`. Unsupported
map literals, `if`/`while`, `break`/`continue` and `defer` are rejected instead of silently
skipping body checking. Use `when` for decisions and the supported list loop/collection
operations. Two historical syntax fixtures now assert rejection. This does not establish
complete concurrency lifecycles, effect isolation or native typed Go lowering.

## Reproducible evidence

- `kiro test examples/typed_errors`: seven integration tests.
- `kiro test examples/quote_service`: twenty domain/handler tests, including 400 and each 422 variant.
- `python3 scripts/probe_variant_evolution.py --kiro ./kiro`: adds a new error variant to
  isolated copies of each quote-service boundary and asserts the expected missing-arm diagnostic.
- Source rejection cases, unchecked runtime cases and native builds cover payload types,
  nil tags, scopes, module identity, capture isolation, schema failures and loop exits.

The mutation probe checks compiler behavior. Agent repair success, token cost, tool calls
and missed handlers still require a controlled experiment with published requirements and
independent acceptance checks; no agent-performance improvement is claimed here.
