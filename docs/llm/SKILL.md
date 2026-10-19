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
- Every value argument must be keyed: `f(name: expression)`, including unary built-ins such as `Ok(value: x)` and `println(value: x)`. Parameterless calls remain `f()`; generic type arguments remain positional.
- Use the declaration's exact parameter names; pass every parameter exactly once, including explicit `nil` for an absent optional. Separate arguments with commas. No positional calls, mixed calls, omitted keys, or argument-name shorthand.
- Expressions evaluate once in source order; keys bind the results to parameters. Never reorder calls' argument expressions while formatting or editing unless changed effect order is intended.
- Read `call_signatures` in the manifest for built-in keys; for example, `test.eq(actual: got, expected: want)` and `json.decode[Type](text: body)`.
- Declare function values as `Fn[(value:i32)->i32]`; include latent effects inside the brackets, e.g. `Fn[(value:str)->nil!io]`. Labels form part of the contract.
- Specialize explicitly with `partial f(key:value)`. Capture expressions and callee evaluation must be pure; values are eagerly snapshotted. Unbound keys and latent effects remain on the new function. Full binding creates a thunk called with `f()`.
- Defaults must be scalar literals in declarations (`amount:i32 = 1`) and explicitly selected (`amount:default`). Omitting the key is still an error. A `Fn` type does not promise defaults.
- Use eager `list.map(items:xs, transform:f)`, `list.filter(items:xs, predicate:p)`, and `list.reduce(items:xs, initial:zero, reduce:r)`. Callbacks must be pure, keyed `value`; reducers use `state` and `value` and return the state type.
- Use `Transform[Input,Output]` for reusable fused plans. Build stages with `list.mapping(transform:f)` and `list.filtering(predicate:p)`, then `list.compose(first:a, then:b)`. Callbacks are pure and snapshotted; construction does not run them.
- Execute a plan eagerly with `list.transduce(items:xs, through:plan, initial:zero, reduce:r)` or materialize only its final list with `list.collect(items:xs, through:plan)`. Each item completes its stages/reduction before the next item. Rejection skips later stages; the first failure aborts. This can expose a different first error from separate eager passes.
- `list.reduce` is an ordered left fold with a required initial value. The old `list.fold` name is rejected; do not add aliases or imply automatic/parallel fusion.
- Use `let` for immutable bindings; the mutable form is `mut name = value`. Partial captures remain stable after rebinding.
- Delimit case dispatch as `when (value) { pattern => { ... } }`; the old unbraced form is rejected.
- Declare only real operational effects: `env`, `fs`, `io`, `log`, `net`, `panic`, `proc`, `time`.
- Use `R[T,E]` for failure, `?` only to propagate `R[T,E]`, `?T` for optionals, and `nil` only for optional or nil-capable returns.
- Use `group`, `spawn`, and `await` for structured concurrency.
- Prefer documented stdlib module names directly: `env`, `fs`, `http`, `json`, `log`, `parse`, `test`, `time`, `list`.
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

## Enforcement boundary

Known expression contracts are checked before builds and again at callable runtime boundaries. This is not a complete type system: record schemas/module-distinct identity, compositional failed-result propagation, broader control flow and full concurrency/effect enforcement remain incomplete. Do not describe these as guaranteed or invent unsupported variants, lambdas, pipeline operators, stateful transducer stages or lazy sequences.
