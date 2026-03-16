# Kiro Syntax Overview (Current frontend subset)

```ki
mod main

import app/router

const Version = "0.4"

type User {
  id:i32
  name:str
  email:?str
}

fn (u:User) display() -> ?str =
  u.name

/// greet returns a greeting.
fn greet(name:str) -> str =
  "hello ${name}"

fn route(path:str) -> Resp {
  group {
    let t = spawn do_work()
    let _ = await t
  }

  return text(200, "ok")
}
```

## Parser support

- `mod <name>`
- `import <name>` and `import <name>/<name>` paths
- `const <Name> = <string|int|ident>` (module scoped)
- `type <Name> { <field>:<type> ... }` (struct form, including optional refs like `?str`)
- `fn <name>(<params>) -> <type> = <body>`
- `fn <name>(<params>) -> <type> { <body> }`
- `fn (<recv>:<Type>) <name>(<params>) -> <type> = <body>`
- `fn (<recv>:<Type>) <name>(<params>) -> <type> { <body> }`
- top-level doc comments (`/// ...`) attached to the next declaration

Function bodies are currently preserved as normalized source text in the AST while the compiler frontend evolves.

## Reserved keywords

Current keywords include: `mod`, `import`, `const`, `type`, `fn`, `let`, `mut`, `if`, `else`, `when`, `for`, `in`, `while`, `break`, `continue`, `defer`, `return`, `spawn`, `await`, `group`.
