# VS Code Extension Packaging Notes

## Language server launch path

The packaged VS Code extension is now wired for the normal user-facing Kiro install flow.

Primary production entrypoint:

- executable: `kiro`
- args: `lsp`

The extension starts `kiro lsp` over stdio and reports a clear error in VS Code if that command cannot be started.

Advanced or developer-only override paths remain available:

- VS Code setting `kiro.lsp.path`
- VS Code setting `kiro.lsp.args`
- environment variable `KIRO_LSP_BIN`

If an override points to a direct `kiro-lsp` binary, the extension leaves arguments empty by default. That path is kept for development and compatibility, but it is not the primary documented production flow.

## Building the `.vsix`

The repository now packages the VS Code extension with:

```bash
./scripts/package_vscode_extension.sh v0.1.0
```

Packaging behavior:

- for tagged release labels, stamps the packaged extension manifest version from the release tag
- validates the vendored VS Code extension runtime files under `editors/vscode/`
- validates the extension manifest
- validates that the default client entrypoint is `kiro lsp`
- writes `dist/kiro-vscode-v0.1.0.vsix`

The extension is intentionally plain JavaScript, so packaging does not require a TypeScript compile step.

## Release and distribution

GitHub Actions now builds the `.vsix` as part of the standard release workflow.

Release automation:

- produces the normal Kiro CLI tarballs
- packages `kiro-vscode-<release>.vsix`
- uploads the `.vsix` as a workflow artifact
- attaches the `.vsix` to tagged GitHub releases
- includes the `.vsix` in release checksums

This keeps the VS Code extension in the same predictable release stream as the CLI bundles.

## Supported user install flow

The supported install flow for normal users is:

1. install `kiro` with the normal release installer
2. download the matching `kiro-vscode-vX.Y.Z.vsix`
3. install the extension from VS Code with **Install from VSIX...**
4. open a Kiro project or any folder with `.ki` files

No manual extension build, `npm install`, `F5`, or `KIRO_LSP_BIN` setup is required for the normal path.

## Known limitations

Current editor capabilities remain limited to the existing Phase 11 scope:

- syntax highlighting
- diagnostics
- formatting
- hover
- go to definition
- document symbols
- basic completion

The extension depends on the `kiro` command being available on `PATH` unless an advanced override is configured.

## Compromises and compatibility notes

- Release bundles still include `kiro-lsp` for compatibility with existing tooling and advanced setups.
- VS Code now treats `kiro lsp` as the primary contract, while other editors can continue to use either `kiro lsp` or a direct `kiro-lsp` binary.
- The workflow avoids GUI automation; validation is packaging- and manifest-focused rather than end-to-end UI testing.
