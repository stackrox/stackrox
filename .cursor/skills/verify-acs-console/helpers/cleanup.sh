#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PID_FILE="${ROOT}/.cursor/skills/verify-acs-console/.launch-ui.pid"

if [[ -f "${PID_FILE}" ]]; then
    pid="$(cat "${PID_FILE}")"
    if kill -0 "${pid}" 2>/dev/null; then
        echo "cleanup: stopping launch-ui pid ${pid}"
        kill "${pid}" 2>/dev/null || true
        sleep 1
        kill -9 "${pid}" 2>/dev/null || true
    fi
    rm -f "${PID_FILE}"
fi

# Only remove transient launch logs; do not delete /opt/cursor/artifacts evidence.
rm -f "${ROOT}/.cursor/skills/verify-acs-console/.launch-ui.log"
echo "cleanup: done (evidence under /opt/cursor/artifacts/verify-acs-console/ preserved)"
