# Config pattern (Phase 7)

Kiro recommends explicit config loading from env.

```ki
type AppConfig {
  port:str
  env:str
}

fn load() -> R[AppConfig, str] {
  let port = env.get_or("PORT", ":8080")
  let app_env = env.get_or("APP_ENV", "dev")
  return Ok(AppConfig{port:port env:app_env})
}
```

Guidelines:

- Keep config parsing close to startup.
- Prefer defaults for non-critical values.
- Return `R[T,E]` for missing/invalid required values.
- Avoid reflection-based config loaders.
