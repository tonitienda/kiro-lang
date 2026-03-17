# Stability policy (Phase 10)

Kiro remains experimental, but Phase 10 defines a clearer stable experimental core.

## Stability tiers

### High stability (stable experimental core)

See `docs/stable_core.md` for the authoritative list.

In short, this includes:

- deterministic formatter and core parser syntax in active use
- module/import/project boundaries
- core control-flow and data constructs
- result/optional flow (`R[T,E]`, `?`, `?T`)
- canonical CLI workflow (`fmt`, `check`, `inspect go`, `new`, `compat`)

### Medium stability (intent stable, details can move)

- diagnostics wording/details
- generated Go organization details
- stdlib helper breadth outside canonical template/example paths

### Lower stability (explicitly experimental)

- placeholder runtime paths for `build/run/test` in this repo slice
- advanced type-system edge cases
- generated Go as a stable external API

## Compatibility expectations

- stable-core fixtures are strongest source-compatibility guardrail
- regressions should be locked with focused fixtures
- experimental fixture sets can evolve faster with clear note updates
