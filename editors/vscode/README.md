# Kiro for Visual Studio Code

The Kiro VS Code extension is packaged as a normal installable `.vsix` artifact.

It provides:

- `.ki` file association and syntax highlighting
- diagnostics from the Kiro parser/check pipeline
- hover, go to definition, and document symbols
- formatting through the canonical `kiro fmt` implementation
- basic completion from the language server

The extension talks to the language server over stdio and uses the supported production entrypoint:

- command: `kiro`
- args: `lsp`

## Install as a normal user

1. Install the `kiro` CLI with the normal installer from a Kiro release.
2. Download the matching VS Code artifact named `kiro-vscode-vX.Y.Z.vsix` from the same release.
3. In VS Code, open **Extensions** and choose **Install from VSIX...**.
4. Open a Kiro project or any folder containing `.ki` files.

If `kiro` is on `PATH`, the extension starts automatically when you open a `.ki` file.

## How the extension finds Kiro

The supported default is to run `kiro lsp` from your `PATH`.

Advanced overrides are available if you need them:

- VS Code setting `kiro.lsp.path`
- environment variable `KIRO_LSP_BIN`
- VS Code setting `kiro.lsp.args` for custom arguments

These overrides are optional and are mainly for development or unusual installations.

## Packaging the `.vsix`

From the repository root:

```bash
./scripts/package_vscode_extension.sh v0.1.0
```

That writes `dist/kiro-vscode-v0.1.0.vsix`. For tagged release labels, the packaging script now stamps the packaged extension manifest version from the tag before building the archive, then validates the manifest and `kiro lsp` entrypoint with repository-local Node checks so CI does not depend on `npm run` behavior.

## Development notes

The extension source still lives in `editors/vscode/`, but users do not need to open that folder or build it manually.

Contributors can package and validate the extension with:

```bash
./scripts/verify_vscode_extension.sh dev
```
