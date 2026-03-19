#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: scripts/verify_install.sh <version> [dist-dir]" >&2
  exit 1
fi

VERSION="$1"
DIST_DIR="${2:-dist}"
INSTALL_ROOT="$(mktemp -d)"
PROJECT_ROOT="$(mktemp -d)"
cleanup() {
  rm -rf "$INSTALL_ROOT" "$PROJECT_ROOT"
}
trap cleanup EXIT

./scripts/write_release_checksums.sh "$VERSION" "$DIST_DIR"
KIRO_INSTALL_BASE_URL="file://$(cd "$DIST_DIR" && pwd)" ./scripts/install.sh --version "$VERSION" --bin-dir "$INSTALL_ROOT/bin"

[[ -x "$INSTALL_ROOT/bin/kiro" ]] || { echo "installed kiro binary missing" >&2; exit 1; }
[[ -x "$INSTALL_ROOT/bin/kiro-lsp" ]] || { echo "installed kiro-lsp binary missing" >&2; exit 1; }
[[ -x "$INSTALL_ROOT/toolchain/go/bin/go" ]] || { echo "installed bundled Go toolchain missing" >&2; exit 1; }

cd "$PROJECT_ROOT"
PATH="$INSTALL_ROOT/bin:/usr/bin:/bin" "$INSTALL_ROOT/bin/kiro" new hello
[[ -f "$PROJECT_ROOT/hello/AGENTS.md" ]] || { echo "scaffolded AGENTS.md missing" >&2; exit 1; }
[[ -f "$PROJECT_ROOT/hello/.kiro/skill/SKILL.md" ]] || { echo "scaffolded SKILL.md missing" >&2; exit 1; }
[[ -f "$PROJECT_ROOT/hello/.kiro/skill/references/kiro.json" ]] || { echo "scaffolded kiro.json missing" >&2; exit 1; }
PATH="$INSTALL_ROOT/bin:/usr/bin:/bin" "$INSTALL_ROOT/bin/kiro" check hello
PATH="$INSTALL_ROOT/bin:/usr/bin:/bin" "$INSTALL_ROOT/bin/kiro" build hello --out "$PROJECT_ROOT/hello-bin"
"$PROJECT_ROOT/hello-bin" > "$PROJECT_ROOT/hello.out"
grep -Fx 'hello' "$PROJECT_ROOT/hello.out"

echo "install verification ok"
