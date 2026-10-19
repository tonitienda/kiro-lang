# Migration notes

Kiro is still pre-1.0 and breaking changes are expected.

## Phase 12: explicit case boundaries

Replace `when value` followed by cases with `when (value) { ... }`. Add one closing brace
after the last case, before any following statement. Nested case blocks each need their
own delimiter. The parser reports a migration hint for the old form; the formatter does
not infer where an unbraced case list was intended to end.

Current shape:

```ki
when (path) {
  "/health" => { return Ok(http.text(200, "ok")) }
  _ => { return Ok(http.not_found()) }
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
