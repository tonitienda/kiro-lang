# Kiro syntax overview

Kiro intentionally keeps the frontend small and regular.

Every call supplies all value arguments by their declared names: `f(name: expression)`.
Arguments require commas; keys may appear in any order, but expressions evaluate in source
order. Parameterless calls use `f()`. Type arguments remain positional, for example
`json.decode[User](text: body)`. See [migration notes](migration.md) for built-in keys.

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
  when (path) {
    "/health" => {
      return Ok(value : http.text(code : 200, body : "ok"))
    }
    _ => {
      return Ok(value : http.not_found())
    }
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

## Explicit functional values

Function types preserve keys: `Fn[(value:i32)->i32]`. Latent effects are inside the type,
e.g. `Fn[(value:str)->nil!io]`. Use `partial add(amount:1)` for deliberate specialization;
ordinary calls still require all keys. Literal defaults (`amount:i32 = 1`) are selected
explicitly with `amount:default`. `let` is immutable; use `mut name = value` for rebinding.
Use `Transform[Input,Output]` for explicit first-then mapping/filtering plans and
`list.transduce` or `list.collect` to execute them. `list.reduce` is the canonical ordered
left fold; `list.fold` is rejected. See [functional-core.md](functional-core.md) for eager
collection helpers and enforcement
limits. The frontend accepts some historical control-flow forms that do not yet execute.

## Removed form

Expression-bodied functions were intentionally removed:

- `fn <name>(...) -> T = expr`

This keeps return flow explicit and simplifies formatting and diagnostics.

## Reserved keywords

Current keywords include: `mod`, `import`, `const`, `type`, `fn`, `let`, `mut`, `if`, `else`, `when`, `for`, `in`, `while`, `break`, `continue`, `defer`, `return`, `spawn`, `await`, `group`, `partial`, `default`.

## Results and data boundaries

`?` requires a result-returning enclosing function and returns Err from that function.
Source record literals require every field explicitly, including optional nils. Typed JSON
decoding validates closed schemas; escaped strings and function/module boundaries preserve
values. See [results-and-schemas.md](results-and-schemas.md) for the implemented semantics.
