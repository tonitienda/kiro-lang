# Phase 11 Notes - Editor Tooling and LSP Baseline

## Implemented in this phase

- Added `kiro-lsp` (`cmd/kiro-lsp`) as a stdio LSP server.
- Added `kiro lsp` CLI convenience command for launching the server.
- Implemented baseline LSP features:
  - initialize/shutdown
  - didOpen/didChange/didClose
  - diagnostics
  - hover
  - go to definition
  - formatting
  - document symbols
  - basic completion
- Added a VS Code extension (`editors/vscode`) with:
  - `.ki` language registration
  - TextMate syntax grammar
  - language configuration
  - LSP client wiring to `kiro-lsp`
- Added editor setup docs for VS Code, Neovim, and Vim (`docs/editor_setup.md`).
- Added editor compatibility fixture samples in `compat/editor/lsp`.
- Added CI workflow for editor tooling checks.
- Hover and symbol signatures now surface explicit function effects.

## Reuse of compiler infrastructure

The LSP implementation still reuses existing compiler layers:

- parsing/lexing are from `internal/parser` and `internal/lexer`
- formatting calls the canonical formatter (`internal/format`)

No second parser/type-checker stack was introduced for editor support.

## Relationship to release/runtime work

The standalone-toolchain phase intentionally preserves editor tooling assumptions:

- `kiro inspect go` remains the debugging backstop for runtime/codegen issues
- the LSP still depends on the same parser/formatter stack, not the runtime path
- release packaging should not introduce editor-specific forks of compiler behavior
- tagged release automation now requires explicit workflow `contents: write` permission so CLI bundles can publish to GitHub Releases without manual intervention
