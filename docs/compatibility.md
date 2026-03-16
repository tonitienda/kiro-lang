# Compatibility corpus and acceptance checks

Phase 8 formalizes a repository-owned compatibility corpus under `compat/`.

## Layout and contract classification

```text
compat/
  syntax/
  cli/
  services/
  concurrency/
  stdlib/
  templates/
  regression/
```

Phase 9 classifies fixture groups by intended compatibility strength:

- **Stable contract fixtures**: `syntax/`, `cli/`, `services/`, `stdlib/`, `templates/`
  - represent source-compatibility-sensitive behavior; drift should be deliberate.
- **Regression fixtures**: `regression/`
  - lock specific bug fixes or diagnostics.
- **Experimental fixtures**: currently `concurrency/`
  - useful for confidence but allowed to evolve faster while semantics continue to harden.

Each fixture is a small directory containing at minimum a `main.ki`. Optional `fixture.json` metadata can declare:

- `expect_success`: whether load/check should pass
- `error_contains`: required diagnostic substring for expected failures
- `modes`: limit fixture execution modes (`fmt`, `check`, `inspect`)
- `inspect_go`: run generated-Go emission check
- `expected_modules`: required generated module files
- `entry`: alternate entry path inside the fixture

## Running locally

```bash
kiro compat
```

or with a custom path/modes:

```bash
kiro compat compat/regression --mode check
```

When compatibility fixtures change intentionally, update `PHASE9_NOTES.md` with rationale.

## What is validated

The compatibility runner currently validates, per fixture:

- formatter idempotence (`fmt(fmt(src)) == fmt(src)`) on all `.ki` files
- parser + project load/import resolution (`check` mode)
- optional generated-Go emission smoke checks (`inspect` mode)
- expected failure diagnostics for regression fixtures

## Coverage goals in this corpus

Current fixtures include representative programs for:

- expression and block function bodies
- `if` / `when`
- mutable locals and reassignment
- collections and interpolation
- struct methods and constants
- `R[T,E]`, `Ok`, `?`, and `?T`/`nil` usage patterns
- loops and control flow (`for`/`while`/`break`/`continue`)
- `defer`
- imports/modules
- doc comment placement
- `spawn`/`await`/`group`
- env/config and service layout patterns
- HTTP hello/JSON service shapes
- `kiro new service` template shape
- generated-Go inspect workflow smoke coverage
- diagnostic regressions (for example unresolved imports, invalid doc comment placement)
