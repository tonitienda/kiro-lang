# Compatibility policy and corpus

Kiro maintains a repository-owned compatibility corpus under `compat/`.

## Fixture categories

```text
compat/
  syntax/
  cli/
  services/
  stdlib/
  templates/
  regression/
  concurrency/
```

Phase 10 compatibility categories:

- **Stable-core fixtures**
  - currently: `syntax/`, `cli/`, `services/`, `stdlib/`, `templates/`
  - strongest source-compatibility promise
- **Compatibility fixtures**
  - long-lived behavior guards that are still expected to evolve occasionally
- **Regression-only fixtures**
  - currently: `regression/`
  - lock bug fixes and diagnostic regressions
- **Diagnostics fixtures**
  - typically represented via `fixture.json` with `expect_success=false` + `error_contains`
- **Template fixtures**
  - currently: `templates/`
  - protect `kiro new` output shape and checks
- **Codegen fixtures**
  - fixtures using `inspect_go`/`expected_modules` assertions
- **Experimental fixtures**
  - currently: `concurrency/`
  - valuable, but allowed to evolve faster while semantics mature

## Fixture metadata

Optional `fixture.json` fields:

- `expect_success`
- `error_contains`
- `modes`
- `inspect_go`
- `expected_modules`
- `entry`

## Running the corpus

```bash
kiro compat
```

or scoped:

```bash
kiro compat compat/regression --mode check
```

## Intentional updates

When intentionally changing fixture behavior:

1. Update fixture source/metadata.
2. Keep category intent clear (stable-core vs regression vs experimental).
3. Update `PHASE10_NOTES.md` (or current active phase notes) with rationale and migration impact.
4. Update docs/templates/examples if user-visible behavior changed.

## What compatibility protects

The corpus validates:

- formatter idempotence on `.ki` files
- parser/project load and module resolution
- optional inspect-go emission checks
- expected diagnostics for failure fixtures
