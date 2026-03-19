#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
usage: scripts/install.sh --version <tag|latest> [--bin-dir <path>]

Install a version-pinned Kiro release bundle for the current platform.

Options:
  --version <tag|latest>  Release tag to install, for example v0.1.0-experimental
  --bin-dir <path>        Directory to install kiro and kiro-lsp into
  -h, --help              Show this help text
USAGE
}

fail() {
  echo "error: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

resolve_default_bin_dir() {
  if [[ -w /usr/local/bin ]]; then
    printf '%s\n' "/usr/local/bin"
  else
    printf '%s\n' "${HOME}/.local/bin"
  fi
}

resolve_os() {
  case "$(uname -s)" in
    Linux) printf '%s\n' "linux" ;;
    Darwin) printf '%s\n' "darwin" ;;
    *) fail "unsupported operating system: $(uname -s); supported values are linux and darwin" ;;
  esac
}

resolve_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf '%s\n' "amd64" ;;
    arm64|aarch64) printf '%s\n' "arm64" ;;
    *) fail "unsupported architecture: $(uname -m); supported values are amd64 and arm64" ;;
  esac
}

sha256_check() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c "$1"
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c "$1"
    return
  fi
  fail "required checksum tool not found: sha256sum or shasum"
}

resolve_latest_version() {
  local api_url="$1"
  local response
  response="$(curl -fsSL "$api_url")" || fail "failed to resolve latest release from ${api_url}"
  local tag
  tag="$(printf '%s' "$response" | tr -d '\n' | sed -E 's/.*"tag_name":"([^"]+)".*/\1/')"
  [[ -n "$tag" && "$tag" != "$response" ]] || fail "could not parse tag_name from ${api_url}"
  printf '%s\n' "$tag"
}

VERSION=""
BIN_DIR="$(resolve_default_bin_dir)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      [[ $# -ge 2 ]] || fail "missing value for --version"
      VERSION="$2"
      shift 2
      ;;
    --bin-dir)
      [[ $# -ge 2 ]] || fail "missing value for --bin-dir"
      BIN_DIR="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

[[ -n "$VERSION" ]] || fail "--version is required"

require_cmd curl
require_cmd tar
require_cmd install
require_cmd mktemp

REPO="${KIRO_INSTALL_REPO:-tonitienda/kiro-lang}"
API_URL="${KIRO_INSTALL_API_URL:-https://api.github.com/repos/${REPO}/releases/latest}"
if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(resolve_latest_version "$API_URL")"
fi

BASE_URL="${KIRO_INSTALL_BASE_URL:-https://github.com/${REPO}/releases/download/${VERSION}}"
OS="$(resolve_os)"
ARCH="$(resolve_arch)"
ARTIFACT="kiro-${VERSION}-${OS}-${ARCH}.tar.gz"
CHECKSUM_FILE="kiro-${VERSION}-checksums.txt"
ARTIFACT_URL="${BASE_URL}/${ARTIFACT}"
CHECKSUM_URL="${BASE_URL}/${CHECKSUM_FILE}"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

ARTIFACT_PATH="${TMP_DIR}/${ARTIFACT}"
CHECKSUM_PATH="${TMP_DIR}/${CHECKSUM_FILE}"
EXTRACT_DIR="${TMP_DIR}/extract"

mkdir -p "$EXTRACT_DIR"

curl -fsSL "$ARTIFACT_URL" -o "$ARTIFACT_PATH" || fail "failed to download ${ARTIFACT_URL}; verify that version ${VERSION} exists for ${OS}/${ARCH}"
curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_PATH" || fail "failed to download ${CHECKSUM_URL}; release checksums are required"

if ! grep -F " ${ARTIFACT}" "$CHECKSUM_PATH" > "${TMP_DIR}/checksum-entry"; then
  fail "checksum file ${CHECKSUM_FILE} does not contain an entry for ${ARTIFACT}"
fi

(
  cd "$TMP_DIR"
  sha256_check "checksum-entry"
) || fail "checksum verification failed for ${ARTIFACT}"

tar -xzf "$ARTIFACT_PATH" -C "$EXTRACT_DIR" || fail "failed to unpack ${ARTIFACT}"
BUNDLE_ROOT="${EXTRACT_DIR}/kiro-${VERSION}-${OS}-${ARCH}"
[[ -d "$BUNDLE_ROOT" ]] || fail "release archive did not contain expected root directory ${ARTIFACT%.tar.gz}"

mkdir -p "$BIN_DIR"
BIN_PARENT="$(cd "$(dirname "$BIN_DIR")" && pwd)"
TOOLCHAIN_DIR="${BIN_PARENT}/toolchain"

install -m 0755 "${BUNDLE_ROOT}/bin/kiro" "${BIN_DIR}/kiro"
if [[ -f "${BUNDLE_ROOT}/bin/kiro-lsp" ]]; then
  install -m 0755 "${BUNDLE_ROOT}/bin/kiro-lsp" "${BIN_DIR}/kiro-lsp"
fi
if [[ -d "${BUNDLE_ROOT}/toolchain" ]]; then
  rm -rf "$TOOLCHAIN_DIR"
  cp -R "${BUNDLE_ROOT}/toolchain" "$TOOLCHAIN_DIR"
fi

INSTALLED_VERSION="$VERSION"
if [[ -f "${BUNDLE_ROOT}/VERSION" ]]; then
  INSTALLED_VERSION="$(tr -d '\n' < "${BUNDLE_ROOT}/VERSION")"
fi

echo "installed kiro ${INSTALLED_VERSION} to ${BIN_DIR}/kiro"
if [[ -f "${BIN_DIR}/kiro-lsp" ]]; then
  echo "installed kiro-lsp to ${BIN_DIR}/kiro-lsp"
fi
if [[ -d "$TOOLCHAIN_DIR/go/bin" ]]; then
  echo "installed bundled Go toolchain to ${TOOLCHAIN_DIR}"
fi
