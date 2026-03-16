# Testing in Kiro

Current repository testing is Go-based for compiler components, with a documented handler-level style for Kiro source.

## Compiler/tooling checks

Run all tests:

```bash
go test ./...
```

This validates lexer, parser, formatter, project resolution, codegen layout, and CLI wiring.

## Handler-level test style

Use request constructors and direct function calls instead of full server boot:

```ki
let req = http.test_req("GET", "/health", "")
let res = app(req)?
test.eq(res.code, 200)
```

See `examples/test_http_handlers`.

## Kiro test command

`kiro test` remains placeholder in this frontend-focused slice, but template/docs are aligned to the expected workflow.
