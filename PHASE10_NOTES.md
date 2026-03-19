# Phase 10 Notes - Stable Core Contract

## Core contract after the standalone-toolchain follow-up

- Core includes language constructs already used by templates/examples and canonical CLI workflow (`fmt`, `check`, `inspect go`, `build`, `run`, `test`, `new`, `compat`).
- Generated Go remains part of the trust/debug story even though executable builds now flow through a standalone generated workdir.
- CLI surface is still conservative; the major change is that `build/run/test` are now real commands instead of placeholders.

## What remains intentionally narrow

- Stable-core does **not** promise a frozen generated-Go API.
- Stable-core does **not** claim all experimental examples execute successfully through the current runtime path.
- Stable-core does promise a boring downstream workflow shape for checking, building, testing, and inspecting projects.

## Follow-up implications

- Release packaging must preserve `inspect go` as the debugging backstop.
- Downstream validation should keep using release bundles, not just source builds.
