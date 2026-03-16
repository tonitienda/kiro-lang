# Project layout

Phase 7 defines clearer service-oriented layout and module boundaries.

## Rules

- Entry can be a directory or a `.ki` file.
- Directory entries must include `main.ki`.
- Project root is the entry directory (or file parent directory).
- Every `.ki` file under root is parsed and checked.
- Imports resolve by:
  - module name (`mod util` supports `import util`)
  - path mapping (`import api/user` -> `api/user.ki` or `api/user/main.ki`)

## Recommended service structure

```text
service/
  main.ki
  app/main.ki
  internal/config/main.ki
  test/health.ki
```

- `main` is composition root.
- `app` contains handlers.
- `internal/config` owns env/config mapping.
- `test` keeps handler-level tests close.

## Generated Go mapping

`kiro inspect go <entry>` writes files to `.kiro-gen` by default:

- source modules under `.kiro-gen/src`
- runtime partition under `.kiro-gen/runtime`
- `x.ki` -> `.kiro-gen/src/x.go`
- `dir/main.ki` -> `.kiro-gen/src/dir/module.go`
