# Phase 10 Notes - Stable Core Contract

Phase 12 retains these core concepts but requires explicit case delimiters,
`when (subject) { ... }`; see `PHASE12_NOTES.md` for the breaking change.

## Core contract after the LLM-oriented simplification pass

- stable-core now assumes **block-only function bodies**
- effects remain explicit and operational
- pseudo-effects such as `!json` are rejected with repair-oriented diagnostics
- result propagation (`?`) stays separate from optionality (`?T`)
- structured concurrency stays explicit with `group`, `spawn`, and `await`
- canonical CLI workflow remains `fmt`, `check`, `inspect go`, `build`, `run`, `test`, `new`, `compat`, `lsp`
- generated Go remains part of the trust/debug story
- the compact `docs/llm/` package is part of the stable-core communication contract for downstream repos and automation
- generated projects now carry a project-local `.kiro/skill/` snapshot, a root `AGENTS.md`, and `.kiro/version.json` by default so that contract can travel with downstream repos

## What remains intentionally narrow

- stable-core does not promise a frozen generated-Go API
- stable-core does not preserve every historical syntax alias or shorthand
- stable-core does promise one boring, canonical style for the documented language slice

Phase 13 extends the keyed core with typed functions, explicit partial application,
visible literal defaults and eager pure collection helpers. See `PHASE13_NOTES.md` and
`docs/functional-core.md` for the implemented contracts and their limits.

Phase 14 renames `list.fold` to `list.reduce` and adds explicit typed transducers.
See `PHASE14_NOTES.md` for ordered execution and allocation limits.
