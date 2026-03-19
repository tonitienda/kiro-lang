# Testing service-style Kiro programs

This guide captures the recommended testing style for small services.

## Testing layers

1. **Pure function tests**: no IO, deterministic inputs/outputs.
2. **Config loader tests**: env parsing and default behavior.
3. **HTTP handler tests**: request/response behavior without starting a real server.
4. **Service smoke tests**: build or run the service and hit `/health`-style endpoints.

## Recommended patterns

### 1) Keep business logic separate from transport

Put transport glue (`http` request/response mapping) in thin handlers, and keep logic in regular functions that are easy to test.

### 2) Test handlers as functions

Prefer direct handler invocation with request fixtures and response assertions instead of starting a full listener for every test.

### 3) Use `kiro test` for module-level tests

```bash
kiro check <project>
kiro test <project>
kiro build <project>
```

### 4) Add at least one release-style smoke test for services

For standalone release validation, verify both:

- the produced binary from `kiro build`
- `kiro run <project>` directly

This repository's release verification script uses the service template and checks `/health` in both modes.
