# Phase 11 Notes - Editor Tooling and LSP Baseline

## Editor/tooling status after the simplification pass

- `kiro-lsp` still reuses the compiler parser/formatter stack
- hover and symbol signatures reflect the canonical block-body surface and explicit effects
- docs/examples/templates now prefer one handler signature style: `fn handler(req:http.Req) -> R[http.Resp, str]`
- diagnostics now use a more repair-oriented shape for missing effects, pseudo-effects, and unresolved imports
- editor setup docs remain valid because syntax changes were normalized in the shared parser/formatter layers
- release bundles and the installer now carry `kiro-lsp` alongside `kiro` so downstream editor setup can pin one release version
- scaffolded projects now vendor a `.kiro/skill/` snapshot, root `AGENTS.md`, and `.kiro/version.json` so downstream repos can hand editors/LLMs a repo-local, version-pinned language bundle with an explicit Codex/agent entry point

## Preserved assumptions

- `kiro inspect go` remains the debugging backstop for runtime/codegen issue
- release packaging should not introduce editor-specific forks of compiler behavior
- tagged release automation now requires explicit workflow `contents: write` permission so CLI bundles and checksum assets can publish to GitHub Releases without manual intervention
- LSP still depends on the same parser and formatter, not a second frontend
- editor tooling continues to surface explicit function effects
