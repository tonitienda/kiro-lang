#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: scripts/package_vscode_extension.sh <version-label> [dist-dir]" >&2
  exit 1
fi

VERSION_LABEL="$1"
DIST_DIR="${2:-dist}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXT_DIR="${REPO_ROOT}/editors/vscode"
PACKAGE_JSON="${EXT_DIR}/package.json"
OUTPUT_FILE="${REPO_ROOT}/${DIST_DIR}/kiro-vscode-${VERSION_LABEL}.vsix"
MANIFEST_VERSION="$(node -p "require('${PACKAGE_JSON}').version")"

if [[ "${VERSION_LABEL}" == v* ]]; then
  EXPECTED_VERSION="${VERSION_LABEL#v}"
  if [[ "${MANIFEST_VERSION}" != "${EXPECTED_VERSION}" ]]; then
    echo "extension manifest version ${MANIFEST_VERSION} does not match release label ${VERSION_LABEL}" >&2
    exit 1
  fi
fi

mkdir -p "${REPO_ROOT}/${DIST_DIR}"

node "${REPO_ROOT}/scripts/verify_vscode_manifest.js"
node "${REPO_ROOT}/scripts/verify_vscode_lsp_entrypoint.js"
python3 "${REPO_ROOT}/scripts/build_vsix.py" "${EXT_DIR}" "${OUTPUT_FILE}"

echo "created ${OUTPUT_FILE}"
