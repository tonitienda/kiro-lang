# Language tour

Current syntax and semantics: [language reference](language-reference.md). This page is
a secondary overview; historical milestones do not define additional executable syntax.

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
  return parse.i32(text : raw)
}

fn maybe_name(req:http.Req) -> ?str {
  return http.query(req : req, key : "name")
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
  let port = env.get_or(key : "PORT", fallback : ":8080")
  return Ok(value : Config{port:port})
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
  return Ok(value : join(a : a, b : b))
}
```

The lifecycle stays explicit:

- create tasks with `spawn`
- synchronize with `await`
- keep task scope visible with `group`
