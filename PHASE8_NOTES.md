# Phase 8 notes

## What changed

- Added a first-class compatibility corpus under `compat/` with syntax, services, concurrency, template, CLI-inspect, and regression fixtures.
- Added a compatibility runner (`kiro compat`) backed by `internal/compat`.
- Added fixture metadata support for expected failure diagnostics and inspect-go assertions.
- Added formatter idempotence checks inside compatibility execution.
- Added CI workflows for core tests, compatibility checks, and examples/templates smoke checks.
- Added contributor/process docs for compatibility and stability.

## Compatibility strategy

- Treat `compat/` fixtures as acceptance tests for source-compatibility-sensitive behavior.
- Keep fixtures small, categorized, and easy to extend.
- Use regression fixtures to lock down diagnostics that should remain stable.

## CI/workflow decisions

- `test.yml`: always runs `go test ./...`.
- `compat.yml`: runs `go run ./cmd/kiro compat`.
- `examples.yml`: validates selected examples and `kiro new` scaffolding with `check/fmt/inspect` smoke checks.

## Known limitations

- Compatibility runner currently focuses on formatter/parser/project-loader and inspect-go smoke checks.
- Backend execution (`kiro run/build/test`) remains placeholder in this slice.
- Diagnostic matching is substring-based (intentionally lightweight).

## Recommended Phase 9 scope

- Expand fixture metadata with expected stdout/exit code once run/build/test pipeline is implemented.
- Add richer semantic invariants and typed diagnostics snapshots when semantic checker matures.
- Increase template/example matrix as command implementations move beyond placeholders.
