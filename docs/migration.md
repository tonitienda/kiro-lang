# Migration notes

Kiro is still pre-1.0 and breaking changes are expected.

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
There are no positional/mixed value calls, default arguments, or `f(name)` shorthand.

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
the actual callable at runtime, before evaluating its arguments. Type compatibility, required
record fields and the other semantic gaps remain separate work.

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
