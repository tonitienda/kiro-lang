# Configuration

Kiro encourages explicit configuration loading through a dedicated module.

## Canonical pattern

```ki
mod config

import env

type AppConfig {
  port:str
  env:str
}

fn load() -> R[AppConfig, str] !env {
  let port = env.get_or(key : "PORT", fallback : ":8080")
  let app_env = env.get_or(key : "APP_ENV", fallback : "dev")
  return Ok(value : AppConfig{port:port env:app_env})
}
```

## Why this pattern is preferred

- environment access is isolated behind `!env`
- startup wiring remains simple
- tests can exercise config loading separately
