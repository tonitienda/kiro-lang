# Testing in Kiro

Current repository testing is Go-based for compiler components.

## Compiler checks

Run all tests:

```bash
go test ./...
```

This validates lexer, parser, formatter, project resolution, and CLI command wiring.

## Kiro test command

`kiro test` exists in CLI surface but remains a placeholder in this frontend-focused slice.
