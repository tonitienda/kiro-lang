# Kiro Syntax Overview (Phase 1 subset)

```ki
mod main

import fs

type Resp {
  code:i32
  body:str
}

fn text(code:i32, body:str) -> Resp =
  Resp{code:code body:body}

fn main() -> i32 =
  let r = text(200, "ok")
  print(r.body)
  0
```

## Current parser support

- `mod <name>`
- `import <name>`
- `type <Name> { <field>:<type> ... }` (struct form)
- `fn <name>(<params>) -> <type> = <body>`

Function bodies are currently stored as normalized source text in the AST during early milestones.
