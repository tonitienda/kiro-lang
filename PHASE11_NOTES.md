# Phase 11 Notes - Editor Tooling and LSP Baseline

## Agent-language research slice (2026-10-19)

- Added `examples/quote_service/`, a seven-module integration seed with 14 domain/handler tests, without changing stable-core syntax.
- Added a research plan and reproducible checker/runtime probes under `docs/research/` and `scripts/probe_agent_contract.py`.
- The audit distinguishes the Go-hosted executable interpreter from structural `inspect go` stubs and records incomplete type, JSON, result and concurrency enforcement. These findings motivate follow-up work; they do not establish an agent-performance advantage.
- The research workflow assumes agent implementation, review, audits and creation of deterministic tools, with no required human code review. Evaluate tooling bootstrapping separately from steady-state application work, including evaluator validation and agent audit errors.

## Editor/tooling status after the simplification pass

- `kiro-lsp` still reuses the compiler parser/formatter stack
- hover and symbol signatures reflect the canonical block-body surface and explicit effects
- docs/examples/templates now prefer one handler signature style: `fn handler(req:http.Req) -> R[http.Resp, str]`
- diagnostics now use a more repair-oriented shape for missing effects, pseudo-effects, and unresolved imports
- editor setup docs remain valid because syntax changes were normalized in the shared parser/formatter layers
- release bundles and the installer now carry `kiro-lsp` alongside `kiro` so downstream editor setup can pin one release version
- the VS Code client now treats `kiro lsp` as the primary production entrypoint and packages as a release `.vsix` for normal users
- VS Code packaging now validates manifest metadata and the `kiro lsp` entrypoint with repository-local Node scripts instead of `npm run`, reducing CI sensitivity to runner npm configuration
- tagged VS Code release packaging now stamps the packaged extension manifest version from the release tag, avoiding CI failures when the checked-in development manifest version has already moved ahead
- scaffolded projects now vendor a `.kiro/skill/` snapshot, root `AGENTS.md`, and `.kiro/version.json` so downstream repos can hand editors/LLMs a repo-local, version-pinned language bundle with an explicit Codex/agent entry point

## Preserved assumptions

- `kiro inspect go` remains the debugging backstop for runtime/codegen issue
- release packaging should not introduce editor-specific forks of compiler behavior
- tagged release automation now requires explicit workflow `contents: write` permission so CLI bundles, the packaged VS Code `.vsix`, and checksum assets can publish to GitHub Releases without manual intervention
- LSP still depends on the same parser and formatter, not a second frontend
- editor tooling continues to surface explicit function effects
