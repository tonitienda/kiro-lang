# Language tour

Kiro is a small server-side language with explicit semantics.

## Declarations

```ki
mod main

const Version = "0.5"

type User {
  id:i32
  name:str
  email:?str
}

fn label(u:User) -> str {
  return "${u.name}#${u.id}"
}
```

## Results and optionals

```ki
fn parse_port(raw:str) -> R[i32, str] {
  return parse.i32(raw)
}

fn maybe_name(req:http.Req) -> ?str {
  return http.query(req, "name")
}
```

- `R[T,E]` is for failure
- `?T` is for absence
- `?` only propagates `R[T,E]`

## Effects

```ki
fn load() -> R[Config, str] !env {
  let port = env.get_or("PORT", ":8080")
  return Ok(Config{port:port})
}
```

## Concurrency

```ki
group {
  let ta = spawn fetch_a()
  let tb = spawn fetch_b()
  let a = await ta
  let b = await tb
  return Ok(http.text(200, "${a}${b}"))
}
```
