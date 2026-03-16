# Project layout

Phase 6 defines explicit project/module resolution for the frontend slice.

## Rules

- Entry can be a directory or a `.ki` file.
- Directory entries must include `main.ki`.
- Project root is the entry directory (or file parent directory).
- Every `.ki` file under the root is parsed and checked.
- Imports resolve by:
  - module name (`mod util` supports `import util`)
  - path mapping (`import api/user` -> `api/user.ki` or `api/user/main.ki`)

## Generated Go mapping

`kiro inspect go <entry>` writes files to `.kiro-gen` by default:

- `x.ki` -> `.kiro-gen/x.go`
- `dir/main.ki` -> `.kiro-gen/dir/module.go`
