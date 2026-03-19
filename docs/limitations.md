# Current limitations

Kiro is still experimental. These limits are intentional to set expectations.

## Backend and portability

- Compilation strategy is still Go-backend first.
- Release bundles currently target Linux/macOS amd64/arm64.
- Building Kiro itself from source still requires Go.
- No browser/JS/WASM target.

## Standalone runtime/toolchain status

- Released Kiro bundles include a bundled Go toolchain so end users do not need to install the `go` CLI separately.
- Source-built `kiro` binaries fall back to a host Go installation unless you point them at a bundled toolchain with `KIRO_GO_BIN` or `KIRO_TOOLCHAIN_DIR`.
- `kiro build`, `kiro run`, and `kiro test` are now real commands, but the execution backend is still a pragmatic subset oriented around the current hello/service/test flows.

## Type system and semantics

- Advanced generic/union edge cases may evolve.
- Result/optional flow is stable in common patterns but not yet proven across large codebases.
- Not every experimental example in the repository is guaranteed to execute successfully through the new standalone runtime path yet.

## JSON and HTTP surface

- JSON decode/validation ergonomics are still evolving.
- HTTP helpers are intentionally small and may refine naming/details.
- Service smoke coverage currently centers on the `/health` style handler workflows used by templates and release validation.

## Diagnostics and generated Go

- Diagnostic quality is improving, but exact wording/location can still change.
- `kiro inspect go` remains the explicit inspection path.
- `kiro build/run/test --keep-gen` preserve the generated execution work directory, but that layout is still experimental and may change between releases.

## Not yet ideal for

- large multi-team codebases requiring strict long-term compatibility guarantees
- projects needing multiple backend targets
- heavy framework/macro-based metaprogramming styles
