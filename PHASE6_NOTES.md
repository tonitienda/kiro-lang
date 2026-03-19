# Phase 6 Notes

## What changed in the original phase

- Added `kiro check <entry-or-path>` for fast parse + import/module validation.
- Added `kiro inspect go <entry-or-path> [--out-dir <dir>]` to emit inspectable Go.
- Added `kiro new <hello|service>` scaffolding command.
- Added source-aware diagnostics for parser failures with line excerpt and caret.
- Added explicit project/module resolution rules and documentation.
- Added production-shaped example directories for service/CLI/file/testing workflows.

## Historical note

The original Phase 6 limitation that `kiro build`, `kiro run`, and `kiro test` were placeholders is no longer true. Those commands are now implemented through the standalone release/runtime path documented in `RELEASE_TOOLCHAIN_NOTES.md`.

## Enduring decisions

- Directory entry requires `main.ki`.
- Imports resolve via declared module names and predictable path mapping.
- `kiro inspect go` remains the explicit inspection path.
