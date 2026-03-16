# Service structure: the Kiro way (Phase 7)

Recommended tiny service layout:

```text
service/
  main.ki
  app/main.ki
  internal/config/main.ki
  test/health.ki
```

## Responsibilities

- `main.ki`: startup flow only.
- `app/*`: handlers and service behavior.
- `internal/config/*`: explicit env/config loading.
- `test/*`: handler-focused tests.

## Startup pattern

1. Load config.
2. Log startup address/environment.
3. Build handler function.
4. Serve HTTP.

This keeps operational behavior obvious and avoids framework gravity.
