# Testing in Kiro

Kiro now has both Go-level compiler tests and an experimental runtime test command.

## Compiler/tooling checks

Run the repository test suite:

```bash
go test ./...
```

This validates lexer, parser, formatter, project resolution, codegen layout, build orchestration, toolchain lookup, and CLI wiring.

## Kiro test command

`kiro test <entry-or-path>` discovers `fn test_*()` functions and runs them through the same generated-Go + native toolchain path used by `kiro build` and `kiro run`.

Example:

```bash
kiro test examples/test_demo
```

Example test style:

```kiro
fn test_add() -> nil {
  test.eq(add(2, 3), 5)
}
```

## Handler-level test style

Use request constructors and direct function calls instead of full server boot when possible:

```kiro
let req = http.test_req("GET", "/health", "")
let res = app(req)?
test.eq(res.code, 200)
```

See `examples/test_http_handlers`.

## Current limits

The test runner is intentionally small in this phase:

- test discovery is prefix-based (`test_*`)
- reporting is readable but minimal
- coverage centers on the current template/service workflows rather than a large custom testing runtime
