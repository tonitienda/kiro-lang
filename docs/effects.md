# Kiro effects

Kiro v1 uses **explicit function effect declarations** for operational, external, or non-deterministic behavior.

## Syntax

Effects appear after the return type in function or method signatures.

```ki
fn load_config() -> R[Config, str] !env
fn main() -> i32 !env !io !log !net {
  let cfg = load_config()?
  log.info("starting ${cfg.port}")
  http.serve(cfg.port, app)?
  return 0
}
```

Functions with no `!effect` markers are treated as pure.

## Built-in effects

Kiro currently recognizes this fixed set of built-in effects:

- `!env`
- `!fs`
- `!io`
- `!log`
- `!net`
- `!panic`
- `!proc`
- `!time`

Unknown effect names are rejected by `kiro check`.

## Core rule

If function `A` calls function `B`, `A` must declare every effect required by `B`.

```ki
fn read_port() -> str !env =
  env.get_or("PORT", ":8080")

fn main() -> i32 !env !io {
  let port = read_port()
  println(port)
  return 0
}
```

This checking is intentionally conservative and explicit.

## What counts as an effect in v1

Effects are for external or operational behavior such as:

- environment access
- filesystem access
- console/stdout/stderr I/O
- logging
- networking
- process interaction
- time access or sleeping
- explicit panic-like runtime escape hatches

## What does **not** count as an effect in v1

Fallible computation is still modeled with `R[T,E]` and `?`.

That means these remain **pure** even when they can fail:

- `json.encode`
- `json.decode`
- parsing/formatting helpers such as `parse.i32`
- serialization/validation that does not itself perform external operations

In other words, **JSON is not an effect** in Kiro v1.

## Canonical formatting

`kiro fmt` canonicalizes effect order lexicographically.

Input:

```ki
fn main() -> i32 !net !env !log {
  return 0
}
```

Formatted output:

```ki
fn main() -> i32 !env !log !net {
  return 0
}
```

## Current limitations

Kiro v1 effect checking is deliberately small:

- no effect polymorphism
- no user-defined effect names
- no effect inference engine
- no effect annotations on variables or blocks
- no special async effect separate from the callee effects used by `spawn`/`await`
