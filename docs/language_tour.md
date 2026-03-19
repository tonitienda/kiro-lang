# Language tour

This tour teaches the canonical Kiro surface.

## Module, type, and function declarations

```ki
mod main

type User {
  id:i32
  name:str
  email:?str
}

fn label(u:User) -> str {
  return "${u.name}#${u.id}"
}
```

## Fallibility vs optionality

```ki
fn parse_port(raw:str) -> R[i32,str] {
  return parse.i32(raw)
}

fn maybe_name(req:http.Req) -> ?str {
  return http.query(req, "name")
}
```

Rules:

- use `R[T,E]` when an operation can fail
- use `?` only to propagate `R[T,E]`
- use `?T` when a value may be absent
- use `nil` only where the type permits absence

## Effects

```ki
fn load() -> R[Config,str] !env {
  let port = env.get_or("PORT", ":8080")
  return Ok(Config{port:port})
}
```

Effects are for operational impurity, not for parsing/JSON/formatting.

## Structured concurrency

```ki
group {
  let ta = spawn fetch_a()
  let tb = spawn fetch_b()
  let a = await ta
  let b = await tb
  return Ok(join(a, b))
}
```

The lifecycle stays explicit:

- create tasks with `spawn`
- synchronize with `await`
- keep task scope visible with `group`
