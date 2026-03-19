# Release toolchain notes

## Chosen strategy

Kiro now uses a **bundled Go toolchain release layout** rather than requiring downstream users to install Go themselves.

### Why this approach

- Kiro remains Go-backed.
- Generated Go stays inspectable.
- Release artifacts can be self-contained and predictable.
- We avoid fragile network downloads during normal `kiro build`, `kiro run`, or `kiro test` usage.

## Release artifact structure

Each release archive is shaped like this:

```text
kiro-vX.Y.Z-<os>-<arch>/
  bin/
    kiro
    kiro-lsp
  toolchain/
    go/
      bin/go
      ...
  README.md
  RELEASE_TOOLCHAIN_NOTES.md
  VERSION
```

Each release also publishes a release-level checksum file:

```text
kiro-vX.Y.Z-checksums.txt
```

The CLI locates its Go toolchain in this order:

1. `KIRO_GO_BIN`
2. `KIRO_TOOLCHAIN_DIR/go/bin/go`
3. `toolchain/go/bin/go` relative to the `kiro` executable
4. `go` on `PATH` as a developer fallback

Each candidate is probed with `go version` before Kiro accepts it. That keeps `kiro build`, `kiro run`, and `kiro test` from failing later with a raw `exec format error` when a bundled toolchain does not match the current host architecture.

## Runtime/build strategy

`kiro build`, `kiro run`, and `kiro test` generate a temporary Go work directory that contains:

- a compact serialized project description
- a small embedded runtime/interpreter kit
- a generated Go entrypoint that builds a native executable

This keeps the normal end-user workflow standalone while preserving a debuggable Go boundary.

## Installer expectation

`scripts/install.sh` installs `kiro` and `kiro-lsp` into the chosen binary directory and places the bundled toolchain in a sibling `toolchain/` directory so the runtime lookup remains deterministic.

Examples:

- `--bin-dir /usr/local/bin` -> `/usr/local/toolchain/go/bin/go`
- `--bin-dir ./bin` -> `./toolchain/go/bin/go`

## Current limitations

- The new runtime path is **pragmatic, not fully feature-complete**. It supports the template/hello/service/test workflows targeted in this phase, but it is not yet a complete implementation of every experimental Kiro example.
- `kiro inspect go` and `kiro build/run/test` do not currently emit the exact same Go layout. `inspect go` remains the explicit source-inspection path, while `build/run/test --keep-gen` preserve the executable work directory used for the standalone toolchain flow.
- Building Kiro itself from source still requires Go.
- Release packaging currently targets Linux/macOS amd64/arm64 bundles; Windows packaging remains out of scope for this phase.

## Follow-up recommendations

1. Converge more of the execution backend and `inspect go` story so one generated-Go model covers both inspection and executable output.
2. Expand runtime coverage across more example programs before widening compatibility claims.
3. Add richer `kiro test` reporting once a more formal test runtime exists.
4. Keep checksum publication, installer behavior, and release asset naming aligned.
