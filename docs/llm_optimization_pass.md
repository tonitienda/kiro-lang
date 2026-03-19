# LLM optimization pass

This document summarizes the current simplification pass.

## What changed

### 1. Function syntax was reduced to one canonical form

Kiro now uses **block-only function bodies**.

Removed form:

```ki
fn add(a:i32, b:i32) -> i32 = a + b
```

Canonical form:

```ki
fn add(a:i32, b:i32) -> i32 {
  return a + b
}
```

### 2. HTTP service examples were standardized

Canonical handler shape:

```ki
fn handler(req:http.Req) -> R[http.Resp, str]
```

This keeps handler fallibility explicit and aligns templates, examples, and docs.

### 3. Effects stayed operational only

The pass reinforces that:

- `env.get_or` is `!env`
- `fs.read_file` is `!fs`
- `http.serve` is `!net`
- `log.info` is `!log`
- `json.encode`, `json.decode`, and `parse.i32` are pure

### 4. Result and optional diagnostics were tightened

`?` now reports an actionable error when used on a non-`R[T,E]` value.

### 5. Concurrency teaching was tightened around lifecycle visibility

`group` is the preferred structured pattern, with `spawn` and `await` kept explicit.

### 6. Stdlib naming was tightened

The small public surface now prefers canonical names such as `fs.read_file` instead of older aliases like `fs.read`.

## Why this improves LLM efficiency

- fewer syntax variants to choose from
- more local reasoning from signatures and blocks
- less ambiguity between effects, failure, and absence
- easier compiler-guided repair loops
- examples, templates, docs, and fixtures now teach the same shapes

## Breaking changes in this pass

- removed expression-bodied function declarations
- standardized service handler signatures on `R[http.Resp, str]`
- removed the `fs.read` alias in favor of `fs.read_file`

## Intentionally unchanged

- effects remain explicit, not inferred
- generated Go stays part of the trust/debug story
- structured concurrency remains explicit rather than magical
