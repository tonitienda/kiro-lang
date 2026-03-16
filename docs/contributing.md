# Contributing and local verification

This repository prefers reliability-first changes.

## Required checks before submitting

Run:

```bash
go test ./...
go run ./cmd/kiro compat
```

## Compatibility fixtures

Add new fixtures under `compat/` grouped by theme. A fixture should be small and explicit.

1. Create a directory with `main.ki`.
2. Add optional `fixture.json` for expected failures or inspect-go assertions.
3. Ensure formatter idempotence and parser load pass via `kiro compat`.

Use regression fixtures for diagnostics you want to lock down.

## Examples and templates

Examples should remain parse/check healthy because CI runs selected checks against `examples/`.

When template output changes (`kiro new hello|service`), update tests/docs and ensure scaffolds pass:

```bash
go run ./cmd/kiro new hello
go run ./cmd/kiro check hello
```

and similarly for `service`.

## If behavior intentionally changes

- update fixtures and/or regression metadata
- update docs (`README.md`, `docs/compatibility.md`, `docs/stability.md`)
- document rationale and migration notes in `PHASE8_NOTES.md`
