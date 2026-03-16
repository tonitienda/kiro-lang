# Kiro Syntax Overview (Current frontend subset)

```ki
mod main

import app/router

type User {
  id:i32
  name:str
}

fn (u:User) display() -> str =
  u.name

fn route(path:str) -> Resp {
  if path == "/health" => {
    return text(200, "ok")
  }

  return text(404, "not found")
}
```

## Parser support

- `mod <name>`
- `import <name>` and `import <name>/<name>` paths
- `type <Name> { <field>:<type> ... }` (struct form)
- `fn <name>(<params>) -> <type> = <body>`
- `fn <name>(<params>) -> <type> { <body> }`
- `fn (<recv>:<Type>) <name>(<params>) -> <type> = <body>`
- `fn (<recv>:<Type>) <name>(<params>) -> <type> { <body> }`

Function bodies are currently preserved as normalized source text in the AST while the compiler frontend evolves.

## Reserved keywords

Current keywords include: `mod`, `import`, `type`, `fn`, `let`, `mut`, `if`, `else`, `when`, `for`, `in`, `while`, `break`, `continue`, `defer`, `return`.
