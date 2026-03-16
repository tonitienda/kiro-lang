# Packages and modules (Phase 7)

Kiro keeps package rules intentionally small and explicit.

## Naming

- Every source file starts with `mod <name>`.
- Module names should be short lowercase identifiers (for example `main`, `app`, `config`).
- Prefer one module identity per directory.

## Structure and entrypoints

- Entrypoint is a directory containing `main.ki` (or an explicit `.ki` file path).
- `main` is the composition root: load config, wire handlers, start serving.
- Keep service logic in non-`main` modules (for example `app`, `internal/config`).

## Import paths

- `import foo` resolves to module `mod foo` under project root.
- `import internal/config` resolves path-style (`internal/config.ki` or `internal/config/main.ki`).
- Keep imports acyclic and explicit; avoid wildcard-style behavior.

## Multiple source files

- Loader reads all `.ki` files under project root.
- `kiro inspect go` preserves relative paths in generated output under `.kiro-gen/src`.
- Prefer splitting by responsibility, not by tiny helper functions.

## Tests and modules

- Handler-level tests should live near service code (`test/*.ki` is recommended).
- Tests should call module functions directly instead of booting full servers.
- Use `kiro test` when runtime lands; meanwhile validate compiler/tooling with `go test ./...`.
