# Service structure

The canonical Kiro service has three layers.

## 1. Entrypoint wiring

`main.ki` owns startup, effect declarations, and server wiring.

```ki
fn main() -> i32 !env !log !net {
  let cfg = config.load()?
  log.info("starting ${cfg.port}")
  http.serve(cfg.port, app.handler)?
  return 0
}
```

## 2. Handler module

`app/main.ki` owns request routing.

```ki
fn handler(req:http.Req) -> R[http.Resp, str] {
  when req.path
    "/health" => {
      return Ok(http.text(200, "ok"))
    }
    _ => {
      return Ok(http.not_found())
    }
}
```

## 3. Config module

`internal/config/main.ki` owns environment-derived configuration.

```ki
fn load() -> R[AppConfig, str] !env {
  let port = env.get_or("PORT", ":8080")
  return Ok(AppConfig{port:port})
}
```

## Testing story

Prefer direct handler tests over booting a real server when possible.
