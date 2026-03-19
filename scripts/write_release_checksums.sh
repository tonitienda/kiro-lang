#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: scripts/write_release_checksums.sh <version> [dist-dir]" >&2
  exit 1
fi

VERSION="$1"
DIST_DIR="${2:-dist}"
OUT_FILE="${DIST_DIR}/kiro-${VERSION}-checksums.txt"

mapfile -t ARTIFACTS < <(find "${DIST_DIR}" -maxdepth 1 -type f \( -name "kiro-${VERSION}-*.tar.gz" -o -name "kiro-vscode-${VERSION}.vsix" \) | sort)
if [[ ${#ARTIFACTS[@]} -eq 0 ]]; then
  echo "no release artifacts found for version ${VERSION} in ${DIST_DIR}" >&2
  exit 1
fi

BASENAMES=()
for artifact in "${ARTIFACTS[@]}"; do
  BASENAMES+=("$(basename "$artifact")")
done

(
  cd "${DIST_DIR}"
  sha256sum "${BASENAMES[@]}" > "$(basename "${OUT_FILE}")"
)

echo "wrote ${OUT_FILE}"
