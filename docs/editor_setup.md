# Editor setup (Phase 11)

Kiro ships a language server and a packaged VS Code extension for normal users.

## Features in scope

Current editor support intentionally focuses on:

- diagnostics (reusing the parser/check pipeline)
- hover on declarations in the current document
- go-to-definition for local and top-level symbols in the current document
- formatting via the canonical `kiro fmt` formatter implementation
- document symbols (outline)
- basic completion
- TextMate syntax highlighting for `.ki` files in VS Code

Deferred features are listed in `PHASE11_NOTES.md`.

## Recommended VS Code workflow

The normal user setup is:

1. install `kiro` from a release with `./scripts/install.sh --version vX.Y.Z`
2. download the matching `kiro-vscode-vX.Y.Z.vsix` release asset
3. in VS Code, use **Extensions → Install from VSIX...**
4. open a Kiro project or a folder containing `.ki` files

Expected features in VS Code:

- syntax highlighting
- diagnostics
- formatting
- hover
- go to definition
- document symbols
- basic completion

## How VS Code finds the language server

The packaged extension uses the supported production entrypoint:

- command: `kiro`
- args: `lsp`

That means a normal user only needs the `kiro` binary on `PATH`.

Advanced overrides exist for unusual setups:

- VS Code setting `kiro.lsp.path`
- VS Code setting `kiro.lsp.args`
- environment variable `KIRO_LSP_BIN`

These are optional and mainly intended for development or compatibility work.

## Packaging the VS Code extension

From the repository root:

```bash
./scripts/package_vscode_extension.sh v0.1.0
```

That produces a normal installable artifact such as `dist/kiro-vscode-v0.1.0.vsix`.

For a release-quality smoke check, run:

```bash
./scripts/verify_vscode_extension.sh v0.1.0
```

## Other editors

The language server stays a stdio server. Other editors can launch either `kiro lsp` or a direct `kiro-lsp` binary.

### Neovim setup (native LSP)

Use `nvim-lspconfig` with `kiro lsp` over stdio.

```lua
local lspconfig = require('lspconfig')

vim.filetype.add({ extension = { ki = 'kiro' } })

lspconfig.kiro = {
  default_config = {
    cmd = { 'kiro', 'lsp' },
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

### Vim setup (coc.nvim)

For Vim users with `coc.nvim`, configure filetype and language server:

```json
{
  "languageserver": {
    "kiro": {
      "command": "kiro",
      "args": ["lsp"],
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
