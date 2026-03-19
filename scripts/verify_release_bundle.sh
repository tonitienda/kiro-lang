#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: scripts/verify_release_bundle.sh <bundle.tar.gz>" >&2
  exit 1
fi

BUNDLE="$1"
WORK_ROOT="$(mktemp -d)"
EXTRACT_ROOT="${WORK_ROOT}/bundle"
PROJECT_ROOT="${WORK_ROOT}/playground"
mkdir -p "${EXTRACT_ROOT}" "${PROJECT_ROOT}"

tar -xzf "${BUNDLE}" -C "${EXTRACT_ROOT}"
BUNDLE_DIR="$(find "${EXTRACT_ROOT}" -mindepth 1 -maxdepth 1 -type d | head -n 1)"
if [[ -z "${BUNDLE_DIR}" ]]; then
  echo "failed to locate extracted bundle root" >&2
  exit 1
fi

KIRO_BIN="${BUNDLE_DIR}/bin/kiro"
if [[ ! -x "${KIRO_BIN}" ]]; then
  echo "bundle does not contain an executable kiro binary" >&2
  exit 1
fi

if command -v go >/dev/null 2>&1; then
  echo "info: host go found at $(command -v go); validation will run with PATH stripped to simulate a standalone consumer"
fi

export PATH="/usr/bin:/bin"
export PORT=":18082"
cd "${PROJECT_ROOT}"

"${KIRO_BIN}" new service
[[ -f "${PROJECT_ROOT}/service/AGENTS.md" ]] || { echo "scaffolded AGENTS.md missing" >&2; exit 1; }
[[ -f "${PROJECT_ROOT}/service/.kiro/skill/SKILL.md" ]] || { echo "scaffolded SKILL.md missing" >&2; exit 1; }
[[ -f "${PROJECT_ROOT}/service/.kiro/skill/references/kiro.json" ]] || { echo "scaffolded kiro.json missing" >&2; exit 1; }
"${KIRO_BIN}" check service
"${KIRO_BIN}" test service
"${KIRO_BIN}" build service --out "${PROJECT_ROOT}/service-bin"

"${PROJECT_ROOT}/service-bin" >"${PROJECT_ROOT}/service.log" 2>"${PROJECT_ROOT}/service.err" &
PID=$!
cleanup() {
  kill "$PID" >/dev/null 2>&1 || true
  wait "$PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

python3 - <<'PY'
import sys, time, urllib.request
url = 'http://127.0.0.1:18082/health'
for _ in range(30):
    try:
        body = urllib.request.urlopen(url, timeout=1).read().decode()
        if body.strip() == 'ok':
            print('service smoke check ok')
            sys.exit(0)
    except Exception:
        time.sleep(0.5)
print('service smoke check failed', file=sys.stderr)
sys.exit(1)
PY

cleanup
trap - EXIT

PORT=":18083" "${KIRO_BIN}" run service >"${PROJECT_ROOT}/kiro-run.log" 2>"${PROJECT_ROOT}/kiro-run.err" &
PID=$!
trap cleanup EXIT
python3 - <<'PY'
import sys, time, urllib.request
url = 'http://127.0.0.1:18083/health'
for _ in range(30):
    try:
        body = urllib.request.urlopen(url, timeout=1).read().decode()
        if body.strip() == 'ok':
            print('kiro run smoke check ok')
            sys.exit(0)
    except Exception:
        time.sleep(0.5)
print('kiro run smoke check failed', file=sys.stderr)
sys.exit(1)
PY
