---
name: kiro-stable-core
description: Use this skill before editing `.ki` files or generating Kiro code. It teaches the stable-core syntax, effects, results/optionals model, canonical stdlib names, project layout, and verification workflow.
---

# Kiro stable-core skill

Use this bundle when a repository contains Kiro source files.

## Read in this order

1. `references/kiro.json` for the compact machine-readable contract.
2. The example closest to the task in `references/examples/`.
3. The repository's root `AGENTS.md` for project-local workflow.

## Stable-core rules

- Use block-only `fn` bodies with explicit `return`.
- Declare only real operational effects: `env`, `fs`, `io`, `log`, `net`, `panic`, `proc`, `time`.
- Use `R[T,E]` for failure, `?` only to propagate `R[T,E]`, `?T` for optionals, and `nil` only for optional or nil-capable returns.
- Use `group`, `spawn`, and `await` for structured concurrency.
- Prefer documented stdlib module names directly: `env`, `fs`, `http`, `json`, `log`, `parse`, `test`, `time`.
- Keep formatting canonical with `kiro fmt`.

## Canonical project conventions

- Tiny CLI projects usually keep `main.ki` at the root.
- Tiny services use `main.ki`, `app/`, `internal/config/`, and `test/`.
- Put startup/effect boundaries in `main.ki`, handlers in `app/`, config loading in `internal/config/`, and handler tests in `test/`.

## Common repair guidance

- Do not invent alternate function forms; rewrite to a block body.
- Do not add pseudo-effects like `!json`; JSON helpers stay pure.
- Do not use `?T` for failure; use `R[T,E]`.
- Do not return `nil` from non-optional types.
- Scope spawned tasks inside a readable `group { ... }` block.

## Verification workflow

After edits, follow the repo guidance and at minimum run:

- `kiro fmt <paths...>`
- `kiro check <entry-or-path>`

For service-shaped projects, also run the documented tests/build commands such as `kiro test .`.
