# Migration notes

Kiro is still pre-1.0 and breaking changes are expected.

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
