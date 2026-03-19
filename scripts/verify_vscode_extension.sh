#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: scripts/verify_vscode_extension.sh <version-label> [dist-dir]" >&2
  exit 1
fi

VERSION_LABEL="$1"
DIST_DIR="${2:-dist}"
OUTPUT_FILE="${DIST_DIR}/kiro-vscode-${VERSION_LABEL}.vsix"

./scripts/package_vscode_extension.sh "${VERSION_LABEL}" "${DIST_DIR}"
[[ -f "${OUTPUT_FILE}" ]] || { echo "missing packaged VS Code extension: ${OUTPUT_FILE}" >&2; exit 1; }

rg -n "command: 'kiro'" editors/vscode/extension.js >/dev/null
rg -n "args: \['lsp'\]" editors/vscode/extension.js >/dev/null
rg -n "Install from VSIX|kiro-vscode-vX\.Y\.Z\.vsix|kiro lsp" README.md docs/editor_setup.md editors/vscode/README.md VSCODE_EXTENSION_PACKAGING_NOTES.md >/dev/null
if rg -n 'Press `F5`|Extension Development Host' README.md docs/editor_setup.md; then
  echo "primary docs still reference the development-host flow" >&2
  exit 1
fi
rg -n "package-vscode-extension|kiro-vscode" .github/workflows/release-toolchain.yml .github/workflows/editor-tooling.yml >/dev/null

echo "vscode extension verification ok"
