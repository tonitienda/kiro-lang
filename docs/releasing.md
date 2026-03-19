# Releasing Kiro bundles

Kiro release artifacts are now **standalone CLI bundles** rather than just raw binaries.

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
  bin/kiro
  toolchain/go/...
  README.md
  RELEASE_TOOLCHAIN_NOTES.md
```

The bundled `toolchain/go` directory is what allows downstream users to run `kiro build`, `kiro run`, and `kiro test` without separately installing the `go` CLI.

## Build release bundles locally

```bash
./scripts/package_release.sh dev 1.22.12 linux amd64
./scripts/package_release.sh dev 1.22.12 linux arm64
./scripts/package_release.sh dev 1.22.12 darwin amd64
./scripts/package_release.sh dev 1.22.12 darwin arm64
```

Artifacts are written to `dist/` using names such as:

- `kiro-dev-linux-amd64.tar.gz`
- `kiro-dev-linux-arm64.tar.gz`
- `kiro-dev-darwin-amd64.tar.gz`
- `kiro-dev-darwin-arm64.tar.gz`

## Validate a release bundle locally

The downstream-style verification script extracts a bundle, strips `PATH` to avoid depending on a host Go installation, scaffolds a service project, and verifies the standalone workflow.

```bash
./scripts/verify_release_bundle.sh dist/kiro-dev-linux-amd64.tar.gz
```

The script checks:

- `kiro new service`
- `kiro check service`
- `kiro test service`
- `kiro build service`
- running the produced service binary and hitting `/health`
- `kiro run service` and hitting `/health`

## GitHub Actions workflow

`.github/workflows/release-toolchain.yml` is the canonical automation for this phase.

It:

1. builds release bundles for all four target tuples
2. uploads them as workflow artifacts
3. verifies the linux/amd64 bundle in a downstream-style smoke test
4. uploads assets to GitHub Releases for tagged `v*` pushes using workflow-level `contents: write` permission

## Pre-release checklist

1. Run `go test ./...`.
2. Run `go run ./cmd/kiro compat`.
3. Verify `kiro build`, `kiro run`, `kiro test`, and `kiro inspect go` locally.
4. Build at least one standalone release bundle with `scripts/package_release.sh`.
5. Run `scripts/verify_release_bundle.sh` against the native bundle.
6. Confirm the release workflow still has `permissions: contents: write` before tagging.
7. Update `README.md`, release docs, limitations, contributing docs, and `RELEASE_TOOLCHAIN_NOTES.md`.
8. Update relevant phase notes before tagging.
