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
- Declare closed variants as `type Error { variant Missing { key:str } variant Offline {} }`; construct with `Error.Missing(key:"sku")` or `Error.Offline()`. Constructors are pure typed callables and support partials.
- Bind payloads as `Error.Missing(key:let key)` or ignore with `key:_`. Pattern calls only destructure; they never invoke functions. All declared variant tags require total explicit arms; no wildcard catch-all. Result matches cover Ok/Err or a final wildcard. Use a nested when for payload decisions.
- Convert error channels explicitly with typed functions and exhaustive matches; `?` never wraps/converts unrelated errors automatically.
- Use `for item in items { ... }` for finite lists. Iteration bindings are immutable and body locals are scoped; outer `mut` updates, nested loops, return and `?` work. Unsupported if/while/break/continue/defer/map literals fail checking.
- Delimit case dispatch as `when (value) { pattern => { ... } }`; the old unbraced form is rejected.
- Declare only real operational effects: `env`, `fs`, `io`, `log`, `net`, `panic`, `proc`, `time`.
- Use `R[T,E]` for failure. `?` requires an enclosing result-returning function and a compatible error type: Err returns from that function, not from its caller. Ok(nil) and Err(nil) are distinct. Runtime/assertion failures are not domain results.
- Migrate main/test functions using `?` to `R[i32,E]` / `R[nil,E]` and return `Ok(value:0)` / `Ok(value:nil)`. Unhandled Err fails process/test boundaries. Use `?T` for optional values, not failure.
- Source record literals must supply every field exactly once, including explicit nil optionals. Unknown/duplicate/missing fields and invalid field types fail. Qualify ambiguous record names by module.
- `json.decode[T](text:body)` requires a schema. JSON rejects missing required, unknown and duplicate fields, wrong nested/list types and trailing data. Omitted optional JSON fields become nil. i32 requires in-range integer text (no decimal/exponent spelling). JSON encoding emits data fields without internal `_type` metadata.
- Preserve escaped string spelling while editing/formatting. Evaluation decodes Go-style escapes once; inserted interpolation data is not executable source.
- Use `group`, `spawn`, and `await` for structured concurrency. Spawn a result-returning function, await its result, then use `?`; do not place inline propagation inside a spawn expression.
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

## Agent discovery and feedback

- Run `kiro describe --json` for language capabilities, diagnostic codes and builtin base contracts; `kiro describe json.decode --json` narrows one signature. Generic/callback specialization still applies.
- Run `kiro symbols . --json` or `kiro symbols . --symbol module.name --json` for declared signatures, defaults, effects, fields, constructors and source revision. This works during semantic repairs but validates syntax only; it is not a resolved call graph.
- Use `kiro check . --json` and inspect code/expected/actual/missing/symbol/hint fields. Checking is first-error; repair and rerun. Broad fallback codes retain legacy messages.
- Respect location_kind: declaration spans identify the enclosing function, not the exact faulty expression. Compiler ranges are one-based Unicode code points; LSP converts to zero-based UTF-16.
- A function whose return type rejects nil must return on every path. A loop may be empty and ? may succeed; add a fallback return. Nil/optional/Any fallthrough remains permitted.

## Optional analysis for maintenance

Expensive analysis is explicitly invoked; do not run every analyzer after every edit or add mandatory hooks without a project requirement. Normal check/lint/LSP commands do not run these tools.

- Before a structural change, consider `kiro analyze graph . --symbol module.fn --json` or `kiro analyze impact . --symbol module.fn --json`. Unknown calls remain visible; an unlisted caller is not proven unaffected.
- For difficult functions, consider `kiro analyze metrics . --symbol module.fn --json` or `kiro analyze slice . --symbol module.fn --value return --json`. Slices select conservative data/control context, not removable code.
- To inspect architecture, use `kiro analyze cycles . --json` and, when useful, `kiro analyze boundaries . --max-nodes 200 --json`. Candidate groups are hypotheses with crossing edges, not automatic refactors or required score targets.
- Enforce policies only on explicit request: `kiro analyze rules . --rules architecture.json --json`. No rules file is auto-discovered. Users choose any CI/hooks.
- For test evidence, use `kiro analyze coverage . --timeout 10s --max-steps 100000 --json`. This executes Kiro tests with fresh interpreters and per-test attribution; effects still execute.
- For assertion quality, explicitly target a covered function: `kiro analyze mutate . --symbol module.fn --max-mutants 10 --total-timeout 60s --json`. Invalid/time-limited mutants are not kills; survivors may be equivalent. `--fail-on-survivor` is an optional user-selected gate.
- Coverage/mutation run in bounded subprocesses; neither is static, neither runs in ordinary check, and neither is a security sandbox. Save full reports outside the source tree and use focused queries to keep context small.

## Verification workflow

After edits, follow the repo guidance and at minimum run:

- `kiro fmt <paths...>`
- `kiro check <entry-or-path> --json`
- `kiro lint <entry-or-path> --json` (semantic contracts, then canonical formatting)
- Use `kiro fmt <paths...> --check --json` for a read-only formatting check.

For service-shaped projects, also run the documented tests/build commands such as `kiro test .`.

## Enforcement boundary

All bodies must parse in the executable grammar. Known expression contracts, closed record/variant schemas, constructor coverage and function-local result propagation are checked before builds; runtime contracts validate actual values. This remains an incremental checker: arbitrary nested-domain exhaustiveness, interprocedural reasoning and full concurrency/effect enforcement remain incomplete. Conservative return-path checking is enforced. Do not invent lambdas, pipeline operators, stateful transducer stages or lazy sequences.
