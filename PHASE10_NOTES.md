# Phase 10 Notes - Stable Core Contract

## Core contract after the LLM-oriented simplification pass

- stable-core now assumes **block-only function bodies**
- effects remain explicit and operational
- pseudo-effects such as `!json` are rejected with repair-oriented diagnostics
- result propagation (`?`) stays separate from optionality (`?T`)
- structured concurrency stays explicit with `group`, `spawn`, and `await`
- canonical CLI workflow remains `fmt`, `check`, `inspect go`, `build`, `run`, `test`, `new`, `compat`, `lsp`
- generated Go remains part of the trust/debug story
- the compact `docs/llm/` package is part of the stable-core communication contract for downstream repos and automation

## What remains intentionally narrow

- stable-core does not promise a frozen generated-Go API
- stable-core does not preserve every historical syntax alias or shorthand
- stable-core does promise one boring, canonical style for the documented language slice
