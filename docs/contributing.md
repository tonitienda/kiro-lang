# Contributing and local verification

This repository prefers reliability-first changes.

## Required checks before submitting

Run:

```bash
go test ./...
go run ./cmd/kiro compat
```

## Phase 9 workflow expectations

When a change is user-visible or architectural, update docs and notes in the same PR:

- `README.md` command/feature surface
- relevant `docs/*` pages
- `PHASE9_NOTES.md` for decisions/tradeoffs
- `AGENTS.md` when contributor/agent workflow guidance changes

## Compatibility fixtures

Add new fixtures under `compat/` grouped by theme and contract strength. A fixture should be small and explicit.

1. Create a directory with `main.ki`.
2. Add optional `fixture.json` for expected failures or inspect-go assertions.
3. Ensure formatter idempotence and parser load pass via `kiro compat`.

Use `compat/regression` fixtures for diagnostics and bug-fix lock-in behavior.

## Generated-Go snapshots and diagnostics regressions

For Phase 9 hardening work, prefer a focused set of stable snapshots/regressions over broad brittle coverage.

- snapshot only representative codegen shapes
- avoid snapshots that couple to irrelevant temp-variable churn
- keep diagnostics assertions stable by signal (`expected/got`, key hint text)

## Examples, mini-projects, and templates

Examples should remain parse/check healthy because CI runs checks against `examples/`.

When template output changes (`kiro new hello|service`), update tests/docs and ensure scaffolds pass:

```bash
go run ./cmd/kiro new hello
go run ./cmd/kiro check hello
```

and similarly for `service`.

As mini-project acceptance coverage is added under `projects/`, wire it into CI and keep project docs aligned.

## If behavior intentionally changes

- update fixtures and/or regression metadata
- update docs (`README.md`, `docs/compatibility.md`, `docs/stability.md`, and relevant new docs)
- document rationale and migration notes in `PHASE9_NOTES.md`
