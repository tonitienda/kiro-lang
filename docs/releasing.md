# Releasing Kiro bundles

Kiro release artifacts are standalone CLI bundles with deterministic names and release-level checksums.

## Supported release targets

Current release automation produces bundles for:

- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64

## Bundle layout

Each archive contains:

```text
kiro-<version>-<os>-<arch>/
  bin/
    kiro
    kiro-lsp
  toolchain/go/...
  README.md
  RELEASE_TOOLCHAIN_NOTES.md
  VERSION
```

The bundled `toolchain/go` directory is what allows downstream users to run `kiro build`, `kiro run`, and `kiro test` without separately installing the `go` CLI.

## Artifact naming

Release artifacts must use these names:

- `kiro-vX.Y.Z-linux-amd64.tar.gz`
- `kiro-vX.Y.Z-linux-arm64.tar.gz`
- `kiro-vX.Y.Z-darwin-amd64.tar.gz`
- `kiro-vX.Y.Z-darwin-arm64.tar.gz`
- `kiro-vX.Y.Z-checksums.txt`
- `kiro-vscode-vX.Y.Z.vsix`

The installer depends on the tarball names, and the editor install flow depends on the VS Code artifact name.

## Build release bundles locally

```bash
KIRO_TOOLCHAIN_SOURCE_DIR="$(go env GOROOT)" ./scripts/package_release.sh dev 1.22.12 linux amd64
KIRO_TOOLCHAIN_SOURCE_DIR="$(go env GOROOT)" ./scripts/package_release.sh dev 1.22.12 linux arm64
KIRO_TOOLCHAIN_SOURCE_DIR="$(go env GOROOT)" ./scripts/package_release.sh dev 1.22.12 darwin amd64
KIRO_TOOLCHAIN_SOURCE_DIR="$(go env GOROOT)" ./scripts/package_release.sh dev 1.22.12 darwin arm64
./scripts/write_release_checksums.sh dev
```

Artifacts are written to `dist/` using names such as:

Using `KIRO_TOOLCHAIN_SOURCE_DIR="$(go env GOROOT)"` reuses the already-installed Go toolchain instead of downloading another copy during local packaging or CI.


- `kiro-dev-linux-amd64.tar.gz`
- `kiro-dev-linux-arm64.tar.gz`
- `kiro-dev-darwin-amd64.tar.gz`
- `kiro-dev-darwin-arm64.tar.gz`
- `kiro-dev-checksums.txt`

## Package the VS Code extension locally

```bash
./scripts/package_vscode_extension.sh v0.1.0
```

For a packaging smoke test plus doc/workflow validation:

```bash
./scripts/verify_vscode_extension.sh v0.1.0
```

The packaged artifact is written to `dist/kiro-vscode-v0.1.0.vsix`.

## Validate a release bundle locally

The downstream-style verification script extracts a bundle, strips `PATH` to avoid depending on a host Go installation, scaffolds a service project, and verifies the standalone workflow.

```bash
./scripts/verify_release_bundle.sh dist/kiro-dev-linux-amd64.tar.gz
```

The script checks:

- `kiro new hello`
- `kiro check hello`
- `kiro build hello`
- `kiro run hello`
- `kiro new service`
- `kiro check service`
- `kiro test service`
- `kiro build service`
- running the produced service binary and hitting `/health`
- `kiro run service` and hitting `/health`

## Validate the installer locally

Use the installer verification script to exercise argument parsing, artifact resolution, checksum verification, and install layout against local release assets.

```bash
./scripts/verify_install.sh dev
```

This script:

- writes `kiro-dev-checksums.txt`
- installs from local `file://` URLs using `scripts/install.sh --version dev`
- verifies `kiro`, `kiro-lsp`, and the bundled toolchain land in the expected locations
- runs the installed `kiro` against a generated hello project

## GitHub Actions workflow

`.github/workflows/release-toolchain.yml` is the canonical automation for this phase.

It:

1. builds release bundles for all four target tuples
2. packages the VS Code extension as `kiro-vscode-<version>.vsix`
3. uploads tarballs and the `.vsix` as workflow artifacts
4. verifies the linux/amd64 bundle in a downstream-style smoke test
5. aggregates release checksums into `kiro-<version>-checksums.txt`, including the `.vsix`
6. verifies the installer flow against the produced artifacts
7. uploads tarballs, the `.vsix`, and the checksum file to GitHub Releases for tagged `v*` pushes using workflow-level `contents: write` permission

## Pre-release checklist

1. Run `go test ./...`.
2. Run `go run ./cmd/kiro compat`.
3. Verify `kiro build`, `kiro run`, `kiro test`, and `kiro inspect go` locally.
4. Build all expected release artifacts with `scripts/package_release.sh`.
5. Run `scripts/write_release_checksums.sh <version>`.
6. Run `scripts/verify_release_bundle.sh` against the native bundle.
7. Run `scripts/verify_install.sh <version>`.
8. Confirm the release workflow still has `permissions: contents: write` before tagging.
9. Package and verify the VS Code extension with `scripts/verify_vscode_extension.sh <version>`.
10. Confirm the release page will include the matching `kiro-vscode-vX.Y.Z.vsix` artifact.
11. Update `README.md`, install/release docs, editor setup docs, contributing docs, `INSTALL_AND_SKILL_NOTES.md`, and `RELEASE_TOOLCHAIN_NOTES.md`.
12. Update relevant phase notes before tagging.
