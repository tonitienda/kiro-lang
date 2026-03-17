# Kiro VS Code extension (Phase 11 baseline)

This extension provides:

- `.ki` language registration
- TextMate syntax highlighting
- language configuration (comments/brackets/autoclose)
- LSP client wiring to `kiro-lsp`

## Development

```bash
cd editors/vscode
npm install
```

Build `kiro-lsp` from repo root:

```bash
go build ./cmd/kiro-lsp
```

Then launch extension development host in VS Code (`F5`).

If `kiro-lsp` is not on `PATH`, set `KIRO_LSP_BIN` in the extension host environment.
