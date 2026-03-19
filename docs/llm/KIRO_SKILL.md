# Kiro Skill

This is the compact, canonical Kiro language package for downstream repos, CI automation, and LLM prompts.

## What Kiro is for

Kiro is an experimental Go-backed server-side language optimized for:

- small CLI programs
- HTTP services
- explicit operational effects
- repairable LLM-generated code
- deterministic formatting and predictable project layout

## What Kiro is not for

Do not treat Kiro as:

- a broad general-purpose language with many equivalent idioms
- a place for effect inference or ambient magic
- a language where failure, optionality, and impurity are interchangeable
- a language with a stable generated-Go API contract

## File extension

- source files use `.ki`

## Stable core at a glance

- module declaration: `mod name`
- imports: `import http`
- block-only functions with explicit `return`
- explicit effects after the return type
- `R[T,E]` for fallible results
- `?` only for propagating `R[T,E]`
- `?T` for optional values
- `nil` only when the type allows absence
- structured concurrency with `group`, `spawn`, `await`
- canonical formatting via `kiro fmt`

## Canonical function declaration

```ki
fn handler(req:http.Req) -> R[http.Resp,str] !net {
  return Ok(http.not_found())
}
```

Rules:

- always use `fn name(params) -> type !effects { ... }`
- always use braces and explicit `return`
- omit `!effects` for pure functions
- keep effect lists small and real; formatting sorts them canonically

## Effects

Effects describe operational impurity only.

Built-in effects:

- `!env`
- `!fs`
- `!io`
- `!log`
- `!net`
- `!panic`
- `!proc`
- `!time`

Semantics:

- pure functions declare no effects
- callers must declare effects required by the functions they call
- pure parsing and encoding helpers are not effects
- `json.encode`, `json.decode`, and `parse.i32` stay pure even when they return `R[T,E]`

## Results, propagation, optionals, and nil

Use these concepts separately:

- `R[T,E]`: the operation can fail with error `E`
- `?`: propagate an `R[T,E]` from the current function
- `?T`: an optional value that may be absent
- `nil`: the absence value for optional or nil-capable types

Canonical examples:

```ki
fn load_port() -> R[str,str] !env {
  return Ok(env.get_or("PORT", ":8080"))
}

fn find_name(id:i32) -> ?str {
  if id == 1 {
    return "kiro"
  }
  return nil
}
```

## Concurrency model

Kiro uses explicit structured concurrency:

- `spawn expr` starts a task
- `await task` waits for that task
- `group { ... }` scopes related concurrent work

Canonical pattern:

```ki
group {
  let left_task = spawn fetch_left()
  let right_task = spawn fetch_right()

  let left = await left_task
  let right = await right_task

  return Ok(Pair{left:left right:right})
}
```

Prefer:

- `group` around related spawned work
- explicit awaits for every spawned task
- local task lifetimes inside one readable block

## Canonical stdlib modules and public API names

Prefer these public module names and helpers in generated code:

- `env`: `get`, `get_or`
- `fs`: `read`, `write`
- `http`: `serve`, `text`, `json`, `not_found`, `test_req`
- `json`: `encode`, `decode`
- `log`: `info`, `error`
- `parse`: `i32`
- `test`: `eq`, `ok`, `fail`
- `time`: `now`

Use the documented module name directly. Do not invent aliases unless a local style guide requires them.

## Canonical project structure

### Tiny CLI

```text
main.ki
```

Canonical shape:

- `main.ki` declares `mod main`
- `fn main() -> i32 !io` or another explicit effect set
- small helpers stay in the same file until structure is clearly needed

### Tiny service

```text
main.ki
app/
  main.ki
internal/
  config/
    main.ki
test/
  health.ki
```

Canonical responsibilities:

- `main.ki`: startup, wiring, effect boundary
- `app/`: request handlers
- `internal/config/`: environment/config loading
- `test/`: handler-level tests

## Canonical testing style

- test functions are named `test_*`
- keep tests explicit and small
- prefer handler tests with `http.test_req`
- use `test.eq` and related helpers for assertions
- return `nil` from tests

Example:

```ki
fn test_health_handler() -> nil {
  let req = http.test_req("GET", "/health", "")
  let res = app.handler(req)?
  test.eq(res.code, 200)
}
```

## Formatting expectations

- run `kiro fmt` on changed paths
- do not preserve alternate spacing styles by hand
- let the formatter choose spaces and effect ordering
- if examples or docs show code, keep it formatter-aligned

## Common mistakes and repairs

| Mistake | Preferred fix |
| --- | --- |
| Writing expression-bodied functions | Use a block body and explicit `return`. |
| Using `!json` | Remove the effect; JSON helpers are pure. |
| Using `?T` where failure is possible | Use `R[T,E]` instead. |
| Using `nil` for a non-optional type | Change the type to `?T` or return a real value. |
| Calling an impure function from a pure function | Add the required effect to the caller. |
| Spawning work without a clear scope | Wrap related tasks in `group { ... }`. |

## Canonical examples

### Hello world

```ki
mod main

fn main() -> i32 !io {
  println("hello")
  return 0
}
```

### Tiny HTTP handler

```ki
mod app

import http

fn handler(req:http.Req) -> R[http.Resp,str] {
  when req.path
    "/health" => {
      return Ok(http.text(200, "ok"))
    }
    _ => {
      return Ok(http.not_found())
    }
}
```

### Config loader

```ki
mod config

import env

fn load_port() -> R[str,str] !env {
  return Ok(env.get_or("PORT", ":8080"))
}
```

## Canonical commands

- `kiro fmt <paths...>`
- `kiro check <entry-or-path>`
- `kiro inspect go <entry-or-path> [--out-dir <dir>]`
- `kiro new <hello|service>`
- `kiro build <entry-or-path> [--out <file>] [--keep-gen]`
- `kiro run <entry-or-path> [--keep-gen] [-- <program args...>]`
- `kiro test <entry-or-path> [--keep-gen]`

## Downstream usage guidance

When a downstream repo wants LLM help:

1. provide this file
2. provide `docs/llm/kiro.json`
3. provide only a few local project files and diagnostics
4. prefer canonical module names, project layout, and formatter output

If syntax or stdlib guidance changes, update this file and `kiro.json` in the same change.
