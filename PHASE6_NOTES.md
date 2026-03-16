# Phase 6 Notes

## What changed

- Added `kiro check <entry-or-path>` for fast parse + import/module validation.
- Added `kiro inspect go <entry-or-path> [--out-dir <dir>]` to emit inspectable Go.
- Added `kiro new <hello|service>` scaffolding command.
- Added source-aware diagnostics for parser failures with line excerpt and caret.
- Added explicit project/module resolution rules and documentation.
- Added production-shaped example directories for service/CLI/file/testing workflows.

## Naming / stdlib consistency decisions

This frontend slice does not execute stdlib behavior yet. Examples and docs now consistently show `env.get_or` and `http.text` patterns while larger stdlib API normalization remains Phase 7 follow-up.

## Project/module decisions

- Directory entry requires `main.ki`.
- Imports resolve via declared module names and predictable path mapping.
- Generated Go mirrors source file layout into `.kiro-gen` (or custom out dir).

## Known limitations

- `kiro build`, `kiro run`, and `kiro test` are still placeholders.
- Generated Go is a debug-oriented stub, not production compilation output.
- Semantic/type diagnostics are limited to parser/import-level checks in this slice.

## Recommended Phase 7 scope

- Implement semantic checking and richer type diagnostics behind `kiro check`.
- Upgrade Go backend from stubs to executable translation.
- Add real `kiro test` runner and output formatting.
- Continue stdlib consistency pass once runtime behavior is active.
