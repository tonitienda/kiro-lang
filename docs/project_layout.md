# Project layout

Kiro now documents one preferred layout for applications and services.

## Small application

```text
app/
  main.ki
```

## Service

```text
service/
  main.ki
  app/
    main.ki
  internal/
    config/
      main.ki
  test/
    health.ki
```

## Responsibilities

- `main.ki`: startup, effect boundaries, wiring
- `app/`: request handlers and pure-ish service logic
- `internal/config/`: environment/config loading
- `test/`: test entry functions and handler-level assertions

## Why this layout

This shape keeps operational boundaries obvious, which improves both human review and model generation.
