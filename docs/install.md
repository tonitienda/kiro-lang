# Installing Kiro releases

Kiro supports two installation stories:

- **build from source** when working inside this repository
- **install a version-pinned release bundle** when a downstream repo or CI job wants a predictable toolchain

## Build from source

```bash
go build ./cmd/kiro
go build ./cmd/kiro-lsp
```

This path is still the right choice for local language development inside `kiro-lang`.

## Install a pinned release bundle

Use the installer script from this repository:

```bash
./scripts/install.sh --version v0.1.0-experimental
./scripts/install.sh --version v0.1.0-experimental --bin-dir ./bin
```

The installer:

- detects the current OS and architecture
- downloads the matching release artifact
- downloads the release checksum file
- verifies the target artifact checksum
- installs `kiro`
- installs `kiro-lsp` when present in the bundle
- installs the bundled Go toolchain next to the chosen binary directory so `kiro build`, `kiro run`, and `kiro test` keep working from a release install

## Supported platforms

The installer accepts release bundles for:

- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64

Unsupported platforms fail clearly.

## Artifact naming convention

Release assets follow a boring predictable shape:

- `kiro-vX.Y.Z-linux-amd64.tar.gz`
- `kiro-vX.Y.Z-linux-arm64.tar.gz`
- `kiro-vX.Y.Z-darwin-amd64.tar.gz`
- `kiro-vX.Y.Z-darwin-arm64.tar.gz`
- `kiro-vX.Y.Z-checksums.txt`

The installer resolves the current platform to one of those names.

## Installed layout

If you install into `--bin-dir /usr/local/bin`, the release install looks like this:

```text
/usr/local/bin/kiro
/usr/local/bin/kiro-lsp
/usr/local/toolchain/go/bin/go
```

If you install into `--bin-dir ./bin`, the layout becomes:

```text
./bin/kiro
./bin/kiro-lsp
./toolchain/go/bin/go
```

That layout matches the runtime toolchain lookup rules used by the CLI.

## CI example

```bash
./scripts/install.sh --version v0.1.0-experimental --bin-dir ./.kiro/bin
PATH="$PWD/.kiro/bin:$PATH" kiro new hello
PATH="$PWD/.kiro/bin:$PATH" kiro check hello
cat hello/AGENTS.md
```

## Notes

- `--version` is the first-class interface; pinned installs are the intended downstream workflow.
- `--version latest` is accepted as a convenience, but explicit tags are preferred for reproducible CI.
- release installs are for consumers; building Kiro itself from source still requires Go.
- `kiro new` from an installed release vendors a project-local skill snapshot and root `AGENTS.md`, both pinned to that installed Kiro version.
