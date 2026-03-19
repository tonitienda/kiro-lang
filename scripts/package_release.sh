#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 4 || $# -gt 5 ]]; then
  echo "usage: scripts/package_release.sh <version> <go-version> <goos> <goarch> [dist-dir]" >&2
  exit 1
fi

VERSION="$1"
GO_VERSION="$2"
TARGET_OS="$3"
TARGET_ARCH="$4"
DIST_DIR="${5:-dist}"
ARTIFACT_NAME="kiro-${VERSION}-${TARGET_OS}-${TARGET_ARCH}"
STAGE_DIR="${DIST_DIR}/${ARTIFACT_NAME}"
TOOLCHAIN_ROOT="${STAGE_DIR}/toolchain"
GO_ARCHIVE="go${GO_VERSION}.${TARGET_OS}-${TARGET_ARCH}.tar.gz"
GO_URL="https://dl.google.com/go/${GO_ARCHIVE}"
CACHE_DIR="${DIST_DIR}/downloads"
KIRO_BIN_NAME="kiro"
KIRO_LSP_BIN_NAME="kiro-lsp"

rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/bin" "${TOOLCHAIN_ROOT}" "${CACHE_DIR}"

LDFLAGS="-X github.com/kiro-lang/kiro/internal/version.KiroVersion=${VERSION}"
GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "${STAGE_DIR}/bin/${KIRO_BIN_NAME}" ./cmd/kiro
GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "${STAGE_DIR}/bin/${KIRO_LSP_BIN_NAME}" ./cmd/kiro-lsp
cp README.md "${STAGE_DIR}/README.md"
cp RELEASE_TOOLCHAIN_NOTES.md "${STAGE_DIR}/RELEASE_TOOLCHAIN_NOTES.md"
printf '%s\n' "${VERSION}" > "${STAGE_DIR}/VERSION"

if [[ -n "${KIRO_TOOLCHAIN_SOURCE_DIR:-}" ]]; then
  rm -rf "${TOOLCHAIN_ROOT}/go"
  cp -R "${KIRO_TOOLCHAIN_SOURCE_DIR}" "${TOOLCHAIN_ROOT}/go"
else
  if [[ ! -f "${CACHE_DIR}/${GO_ARCHIVE}" ]]; then
    curl -fsSL "${GO_URL}" -o "${CACHE_DIR}/${GO_ARCHIVE}"
  fi
  tar -xzf "${CACHE_DIR}/${GO_ARCHIVE}" -C "${TOOLCHAIN_ROOT}"
fi
chmod +x "${STAGE_DIR}/bin/${KIRO_BIN_NAME}" "${STAGE_DIR}/bin/${KIRO_LSP_BIN_NAME}" "${TOOLCHAIN_ROOT}/go/bin/go"

tar -C "${DIST_DIR}" -czf "${DIST_DIR}/${ARTIFACT_NAME}.tar.gz" "${ARTIFACT_NAME}"
echo "created ${DIST_DIR}/${ARTIFACT_NAME}.tar.gz"
