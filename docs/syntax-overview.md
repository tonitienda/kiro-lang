# Kiro syntax overview

Kiro intentionally keeps the frontend small and regular.

## Canonical example

```ki
mod main

import http

const Version = "0.5"

type User {
  id:i32
  name:str
  email:?str
}

fn (u:User) display() -> str {
  return u.name
}

fn greet(name:str) -> str {
  return "hello ${name}"
}

fn route(path:str) -> R[http.Resp, str] {
  when path
    "/health" => {
      return Ok(http.text(200, "ok"))
    }
    _ => {
      return Ok(http.not_found())
    }
}
```

## Supported declaration forms

- `mod <name>`
- `import <name>` and `import <name>/<name>`
- `const <Name> = <string|int|ident>`
- `type <Name> { <field>:<type> ... }`
- `fn <name>(<params>) -> <type> { <body> }`
- `fn <name>(<params>) -> <type> !effect... { <body> }`
- `fn (<recv>:<Type>) <name>(<params>) -> <type> { <body> }`
- `fn (<recv>:<Type>) <name>(<params>) -> <type> !effect... { <body> }`
- top-level doc comments (`/// ...`) attached to the next declaration

## Removed form

Expression-bodied functions were intentionally removed:

- `fn <name>(...) -> T = expr`

This keeps return flow explicit and simplifies formatting and diagnostics.

## Reserved keywords

Current keywords include: `mod`, `import`, `const`, `type`, `fn`, `let`, `mut`, `if`, `else`, `when`, `for`, `in`, `while`, `break`, `continue`, `defer`, `return`, `spawn`, `await`, `group`.
