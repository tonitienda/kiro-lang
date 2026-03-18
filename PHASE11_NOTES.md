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

The LSP implementation reuses existing compiler layers:

- parsing/lexing are from `internal/parser` and `internal/lexer`
- formatting calls the canonical formatter (`internal/format`)

No second parser/type-checker stack was introduced for editor support.

## Current limitations (intentional)

- Hover/definition/completion are document-scoped and declaration-oriented.
- No workspace-wide references, rename, code actions, inlay hints, or semantic tokens.
- No debugger/DAP integration.
- No editor-specific deep plugin ecosystem beyond baseline VS Code + standard LSP client docs.

## Recommended next step

Before experimental release packaging, tighten symbol-resolution fidelity for cross-file/workspace references while preserving the current single-source-of-truth compiler pipeline.
