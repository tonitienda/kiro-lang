# Kiro effects

Kiro uses **explicit function effect declarations** for operational, external, or non-deterministic behavior.

## Canonical syntax

Effects appear after the return type in function or method signatures.

```ki
fn load_config() -> R[Config, str] !env {
  let port = env.get_or("PORT", ":8080")
  return Ok(Config{port:port})
}

fn main() -> i32 !env !log !net {
  let cfg = load_config()?
  log.info("starting ${cfg.port}")
  http.serve(cfg.port, app.handler)?
  return 0
}
```

Functions with no `!effect` markers are pure.

## Built-in effects

- `!env`
- `!fs`
- `!io`
- `!log`
- `!net`
- `!panic`
- `!proc`
- `!time`

Unknown effect names are rejected.

## Core rule

If function `A` calls function `B`, `A` must declare every effect required by `B`.

```ki
fn read_port() -> str !env {
  return env.get_or("PORT", ":8080")
}

fn main() -> i32 !env !io {
  let port = read_port()
  println(port)
  return 0
}
```

## What counts as an effect

Effects are for operational behavior such as:

- environment access
- filesystem access
- console I/O
- logging
- networking
- process interaction
- time access or sleeping

## What is not an effect

Fallibility is **not** an effect.

These remain pure even though they may return `R[T,E]`:

- `json.encode`
- `json.decode`
- `parse.i32`
- validation/serialization that does not touch external state

## Formatter behavior

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

## Intentional limitations

- no effect polymorphism
- no user-defined effects yet
- no effect inference
- no block-local effect annotations
