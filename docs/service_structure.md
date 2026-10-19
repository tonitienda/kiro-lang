# Service structure

Kiro services should use one boring, explicit shape.

## main.ki owns the shell

```ki
mod main

import app
import internal/config
import http
import log

fn main() -> i32 !env !log !net {
  let cfg = config.load()?
  log.info(message : "starting ${cfg.port}")
  http.serve(addr : cfg.port, handler : app.handler)?
  return 0
}
```

## app/ owns handlers

```ki
mod app

import http

fn handler(req:http.Req) -> R[http.Resp,str] {
  when (req.path) {
    "/health" => {
      return Ok(value : http.text(code : 200, body : "ok"))
    }
    _ => {
      return Ok(value : http.not_found())
    }
  }
}
```

## internal/config owns env reads

```ki
fn load() -> R[AppConfig,str] !env {
  let port = env.get_or(key : "PORT", fallback : ":8080")
  return Ok(value : AppConfig{port:port})
}
```

This pattern keeps the impure shell thin and keeps handlers easy to test.
