# Stability model

Kiro is experimental, but not intentionally chaotic.

## What we try to keep stable

- documented syntax and formatting behavior
- module/project resolution rules
- canonical CLI workflow (`fmt`, `check`, `inspect go`, `build`, `run`, `test`, `new`, `compat`)
- generated-Go inspection as part of the debugging model

## What may still change faster

- runtime coverage and backend implementation details
- release packaging internals
- the exact generated Go used by standalone executable builds
- diagnostic wording for newer language/runtime paths
