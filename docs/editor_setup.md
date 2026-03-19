# Editor setup (Phase 11)

Kiro now ships a small language-server implementation and a basic VS Code extension.

## Features in scope

Current editor support intentionally focuses on:

- diagnostics (reusing parser/check pipeline)
- hover on declarations in the current document
- go-to-definition for local/top-level symbols in the current document
- formatting via canonical `kiro fmt` formatter implementation
- document symbols (outline)
- basic completion

Deferred features are listed in `PHASE11_NOTES.md`.

## Install `kiro-lsp`

### Build from source

```bash
go build ./cmd/kiro-lsp
```

This produces a `kiro-lsp` binary in the current directory.

### Install from a release bundle

Pinned release installs also install `kiro-lsp` when it is present in the release artifact:

```bash
./scripts/install.sh --version v0.1.0-experimental
./scripts/install.sh --version v0.1.0-experimental --bin-dir ./bin
```

That is the recommended path for downstream repos and editor-specific setup guides that want a stable version pin.

## VS Code setup

Extension source lives in `editors/vscode/`.

### Development install

1. Install `kiro-lsp` from a release bundle or build it from source and make it available on `PATH`.
2. If `kiro-lsp` is not on `PATH`, set `KIRO_LSP_BIN` in the VS Code extension host environment.
3. Open `editors/vscode` in VS Code.
4. Run `npm install` in that folder.
5. Press `F5` (Run Extension) to start an Extension Development Host.
6. Open a `.ki` file.

You should get syntax highlighting, diagnostics, formatting, hover, definition, symbols, and basic completion.

## Neovim setup (native LSP)

Use `nvim-lspconfig` with `kiro-lsp` over stdio.

```lua
local lspconfig = require('lspconfig')

vim.filetype.add({ extension = { ki = 'kiro' } })

lspconfig.kiro = {
  default_config = {
    cmd = { 'kiro-lsp' },
    filetypes = { 'kiro' },
    root_dir = function(fname)
      return lspconfig.util.root_pattern('main.ki', '.git')(fname) or vim.loop.cwd()
    end,
  },
}

lspconfig.kiro.setup({})
```

Optional formatting keybind:

```lua
vim.keymap.set('n', '<leader>kf', function()
  vim.lsp.buf.format({ async = true })
end)
```

## Vim setup (coc.nvim)

For Vim users with `coc.nvim`, configure filetype and language server:

```json
{
  "languageserver": {
    "kiro": {
      "command": "kiro-lsp",
      "filetypes": ["kiro"]
    }
  }
}
```

Add in vimrc:

```vim
autocmd BufRead,BufNewFile *.ki set filetype=kiro
```

## Consistency with CLI

Editor diagnostics and formatting are intentionally aligned with CLI behavior:

- diagnostics come from the same parse/check stack used by `kiro check`
- formatting uses the same formatter implementation used by `kiro fmt`
- generated-Go inspection (`kiro inspect go`) remains the debugging backstop for runtime/codegen investigation
