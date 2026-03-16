# Kiro Syntax Overview (Current frontend subset)

```ki
mod main

import app/router

type Resp {
  code:i32
  body:str
}

fn text(code:i32, body:str) -> Resp =
  Resp{code:code body:body}

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

Function bodies are currently preserved as normalized source text in the AST while the compiler frontend evolves.
