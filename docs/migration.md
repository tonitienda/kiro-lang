# Migration notes

Kiro is still pre-1.0 and breaking changes are expected.

## Phase 15: compositional failures and strict schemas

`?` now requires an enclosing `R[T,E]` function with a compatible error channel. Migrate
main/test functions that unwrap results to `R[i32,E]` / `R[nil,E]` and wrap success returns
with Ok. A failed unwrap returns Err from that function; unhandled process/test Err values
fail, including Err(nil). Spawn a result-returning function and propagate after awaiting.

Record constructors require every declared key, including optional keys. JSON decoding
requires an explicit schema, rejects unknown/duplicate/missing required fields and wrong
types, and fills omitted optionals with nil. Projection schemas that previously ignored
extra JSON fields must become complete schemas. JSON no longer emits runtime `_type`
metadata; result tags remain explicit. i32 text must be an in-range integer without a
fraction/exponent spelling. Strings support standard Go escapes; escape literal backslashes.

See [the complete contract](results-and-schemas.md). Refresh guides for
`phase15-results-schemas`.

## Phase 14: canonical reduce and explicit fusion

Replace `list.fold(items:xs, initial:zero, reduce:r)` with
`list.reduce(items:xs, initial:zero, reduce:r)`. The keys and ordered left-fold semantics
are unchanged. Old calls and aliases receive a migration diagnostic.

For fused processing, create a `Transform[T,U]` using `list.mapping`, `list.filtering`,
and `list.compose(first:a, then:b)`. Execute it with `list.transduce` or `list.collect`.
Separate map/filter calls remain eager and materialized. Explicit transducers execute
per item, so their first error may differ from separate full passes; see
[the functional contract](functional-core.md). Regenerate guides for `phase14-transducers`.

## Phase 13: explicit functional values

See [the functional core](functional-core.md) for `Fn` contracts, `partial`, defaults and
pure collection helpers. Required keyed calls remain unchanged. Select literal defaults
with `key:default`; no key may be omitted. `let` reassignment now fails; use the existing
`mut name = value` form. Known value/return mismatches and hidden effects through typed
aliases now fail checking or callable runtime validation. Numeric strings are no longer
coerced during arithmetic. Regenerate vendored guides for `phase13-functional-core`.

## Phase 12: required named arguments

Every value argument must use the corresponding parameter name, including unary calls:

```ki
fn join(left:str, right:str) -> str {
  return left + right
}

fn main() -> i32 !io {
  println(value: join(right: "B", left: "A"))
  return 0
}
```

Parameterless calls remain `f()`. Generic type arguments remain positional. Named values
require commas (an optional trailing comma is accepted), and every parameter must appear
exactly once. Optional values still require an explicit keyed value such as `coupon: nil`.
There are no positional/mixed value calls or `f(name)` shorthand. Phase 13 adds explicit
`key:default` selection without relaxing the required-key rule.

Calls bind by name, but evaluate expressions exactly once in source order. Reordering keys
is safe for pure values; moving effectful expressions may change behavior. The formatter
preserves source order. Renaming a parameter is now a callable API change. Method receivers
remain implicit in `value.method(name: expression)`.

Common built-in signatures (complete keys are in `docs/llm/references/kiro.json`):

| Function | Required keys |
| --- | --- |
| `print`, `println`, `Ok`, `json.encode` | `value` |
| `Err` | `error` |
| `test.eq` | `actual`, `expected` |
| `env.get_or` | `key`, `fallback` |
| `log.info` | `message` |
| `parse.i32`, `json.decode[T]` | `text` |
| `fs.read_file` | `path` |
| `fs.write_file` | `path`, `contents` |
| `http.text`, `http.json` | `code`, `body` |
| `http.serve` | `addr`, `handler` |
| `http.test_req` | `method`, `path`, `body` |
| `http.query` | `req`, `key` |

`print` and `println` each take one value. Records use `Type{field:value}` construction;
the former empty/type-call runtime shortcut is rejected. The stale file-report example now
uses the implemented `fs.read_file` helper and unwraps its result.

The checker validates named-call syntax throughout accepted bodies and checks key sets for
directly resolved functions, supported built-ins and methods whose receiver type it can
resolve. Aliases/higher-order calls and otherwise unresolved signatures are checked against
the actual callable at runtime, before evaluating its arguments. Phase 13 adds known type and higher-order contract checking. Required record fields and
other documented semantic gaps remain separate work.

## Phase 12: explicit case boundaries

Replace `when value` followed by cases with `when (value) { ... }`. Add one closing brace
after the last case, before any following statement. Nested case blocks each need their
own delimiter. The parser reports a migration hint for the old form; the formatter does
not infer where an unbraced case list was intended to end.

Current shape:

```ki
when (path) {
  "/health" => { return Ok(value : http.text(code : 200, body : "ok")) }
  _ => { return Ok(value : http.not_found()) }
}
```

Rebuild the compiler and regenerate project-local `.kiro/skill/` snapshots to use the new
grammar. Existing snapshots are version-pinned and do not update themselves. See the
[agent-only reassessment](research/agent_core_reassessment.md) for rationale and the distinction
between this implemented change and later type/verification proposals.

## Current redesign direction

### Removed or rejected

- expression-bodied functions
- pseudo-effects like `!json`
- mixed handler signatures in docs/examples/templates
- multiple service/testing styles in public docs

### Canonical replacements

- use block bodies with explicit `return`
- use `R[T,E]` for pure fallible transforms
- use `fn handler(req:http.Req) -> R[http.Resp,str]`
- use `group` / `spawn` / `await` for visible structured concurrency

## Phase 16: variants, matching and executable bodies

Replace stringly domain errors with explicitly declared keyed variant constructors where
useful. Bind payloads with `let` in patterns. Match every declared variant with a total
arm; replace variant wildcards with explicit arms, and use nested matches for payload
decisions. Convert unrelated error channels explicitly before propagating with `?`.

Every body must now parse in the executable grammar; unsupported maps, if/while,
break/continue and defer no longer pass checking silently. Use when and finite list for
loops/collection operations. Branch and loop locals do not escape their scope.
See [the contract](variants-and-matching.md). Regenerate vendored guide snapshots.

## Phase 17: feedback tools and return paths

Non-nil function contracts must return on every successful path. Add a fallback after
loops or successful ? operations; primitive dispatch needs complete exit coverage.
Use `kiro check . --json` for structured feedback, `kiro describe --json` and `kiro symbols . --json` for discovery, and `kiro lint . --json` for semantic plus canonical-format validation.
Formatter directory traversal now skips hidden directories; check mode never writes.
See [agent tools](agent-tools.md) and regenerate guide snapshots.
