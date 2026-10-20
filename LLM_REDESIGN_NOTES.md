# LLM Redesign Notes

This file records the aggressive redesign pass that sharpens Kiro for LLM-generated and LLM-maintained code.

Phase 12 revisits these design assumptions for agent-only implementation/review/audit.
`when (subject) { ... }` and named value calls (`f(name: expression)`) are now required; see `PHASE12_NOTES.md` and
`docs/research/agent_core_reassessment.md`. Later tagged-value/type proposals are not yet implemented.

## Redesign goals

The redesign optimized for:

- predictability
- regularity
- semantic clarity
- compiler-guided repair
- canonical formatting
- low ambiguity
- small explicit surface area

## Breaking changes and tightened rules

### 1. One canonical function form

Removed historical tolerance for expression-bodied functions. Kiro now teaches block-only functions with explicit `return`.

### 2. Effects are strictly operational

Effects are now documented and diagnosed as a small built-in set only:

- `env`
- `fs`
- `io`
- `log`
- `net`
- `panic`
- `proc`
- `time`

Pseudo-effects such as `!json` are rejected with a repair hint explaining that JSON is pure and should use `R[T,E]`.

### 3. Fallibility and optionality are separated more aggressively

The canonical guidance is now:

- `R[T,E]` for failure
- `?` only for propagating `R[T,E]`
- `?T` for optional values
- `nil` only where the type allows absence

### 4. One canonical service shape

Docs, templates, and compatibility fixtures now converge on:

- shell in `main.ki`
- handlers in `app/`
- config loading in `internal/config/`
- handler-level tests in `test/`

### 5. Compatibility corpus reclassified around the new language

New fixture buckets:

- `stable_core`
- `diagnostics`
- `regression`

The goal is to protect the redesigned language, not preserve historical syntax accidents.

## Diagnostic redesign

Project diagnostics now prefer a stable repair-oriented shape:

- what happened
- what was expected
- hint for the next edit

This pass tightened wording for:

- missing effect declarations
- unknown effects, especially pseudo-effects like `!json`
- duplicate effects
- unresolved imports

## Canonical patterns after the redesign

### Function signature

```ki
fn load() -> R[Config,str] !env {
  let port = env.get_or(key : "PORT", fallback : ":8080")
  return Ok(value : Config{port:port})
}
```

### Handler

```ki
fn handler(req:http.Req) -> R[http.Resp,str] {
  return Ok(value : http.not_found())
}
```

### Structured concurrency

```ki
group {
  let t = spawn work()
  let v = await t
  return Ok(value : v)
}
```

## Intentionally experimental

These areas remain intentionally narrow or unfinished:

- deeper semantic checking beyond current parser/project validation
- broader concurrency diagnostics inside function bodies
- generated-Go API stability
- stdlib breadth beyond the documented canonical modules

Phase 13 extends the keyed core with typed functions, explicit partial application,
visible literal defaults and eager pure collection helpers. See `PHASE13_NOTES.md` and
`docs/functional-core.md` for the implemented contracts and their limits.

Phase 14 renames `list.fold` to `list.reduce` and adds explicit typed transducers.
See `PHASE14_NOTES.md` for ordered execution and allocation limits.

Phase 15 implements compositional result propagation and closed record/JSON schemas.
See `PHASE15_NOTES.md` and `docs/results-and-schemas.md` for current enforcement and limits.

Phase 16 adds typed variants, exhaustive recovery, explicit error conversion and executable
list loops, and rejects unparseable bodies during checking; see `PHASE16_NOTES.md`.

Phase 17 adds structured compiler feedback, discovery commands, formatter checks, conservative
return-path analysis and semantic LSP overlays; see `PHASE17_NOTES.md` and `docs/agent-tools.md`.
