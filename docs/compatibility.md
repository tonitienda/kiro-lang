# Compatibility corpus

Kiro uses compatibility fixtures to protect the intended core language and workflow.

## What the corpus is for

Fixtures are used to protect:

- canonical formatting
- parser + project loading behavior
- diagnostics for important mistakes
- template shape
- inspect-go/codegen regressions

## Current fixture themes

The repository is converging on these categories:

- `compat/syntax/` — stable core syntax
- `compat/services/` — canonical service shapes
- `compat/templates/` — `kiro new` outputs
- `compat/cli/` — inspect-go and CLI workflow checks
- `compat/concurrency/` — structured concurrency patterns
- `compat/diagnostics/` and `compat/regression/` — repair-focused failures and regressions

## Optimization-pass focus

The corpus now prioritizes the **new canonical core**, not preservation of every historical syntax form.

That means obsolete forms should move into diagnostics fixtures only when they still help repair loops.
