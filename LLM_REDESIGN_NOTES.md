# LLM Redesign Notes

This file records the aggressive redesign pass that sharpens Kiro for LLM-generated and LLM-maintained code.

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
  let port = env.get_or("PORT", ":8080")
  return Ok(Config{port:port})
}
```

### Handler

```ki
fn handler(req:http.Req) -> R[http.Resp,str] {
  return Ok(http.not_found())
}
```

### Structured concurrency

```ki
group {
  let t = spawn work()
  let v = await t
  return Ok(v)
}
```

## Intentionally experimental

These areas remain intentionally narrow or unfinished:

- deeper semantic checking beyond current parser/project validation
- broader concurrency diagnostics inside function bodies
- generated-Go API stability
- stdlib breadth beyond the documented canonical modules
