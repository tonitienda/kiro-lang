# Testing service-style Kiro programs

This guide captures the recommended testing style for small services.

## Testing layers

1. **Pure function tests**: no IO, deterministic inputs/outputs.
2. **Config loader tests**: env parsing and default behavior.
3. **HTTP handler tests**: request/response behavior without starting a real server.
4. **Service logic tests**: orchestration and `R[T,E]` failure paths.

## Recommended patterns

### 1) Keep business logic separate from transport

Put transport glue (`http` request/response mapping) in thin handlers, and keep logic in regular functions that are easy to test.

### 2) Test handlers as functions

Prefer direct handler invocation with request fixtures and response assertions (`status`, headers, JSON body), instead of starting a full listener.

### 3) Cover failure paths explicitly

When using `R[T,E]`:

- test `Ok` path
- test `Err` path
- assert the external mapping (for example, `Err` -> `bad_request` or `not_found`) remains stable

### 4) Keep fixtures tiny and local

Use small input payloads and explicit expected values. Avoid hidden global setup.

## Suggested directory shape

```text
service/
  main.ki
  config.ki
  handlers.ki
  handlers_test.ki
  app.ki
```

## CI expectations

Service-like examples and templates should pass:

```bash
kiro fmt <project>
kiro check <project>
kiro test <project>
kiro build <project>
```

For now, if command implementations are placeholders, keep tests centered on parser/check/compat + inspect-go workflows until runtime execution is fully wired.
