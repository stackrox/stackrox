#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PLATFORM="${ROOT}/ui/apps/platform"
PID_FILE="${ROOT}/.cursor/skills/verify-acs-console/.launch-ui.pid"
LOG_FILE="${ROOT}/.cursor/skills/verify-acs-console/.launch-ui.log"

if [[ -f "${PID_FILE}" ]] && kill -0 "$(cat "${PID_FILE}")" 2>/dev/null; then
    echo "launch-ui: already running (pid $(cat "${PID_FILE}"))"
    exit 0
fi

cd "${PLATFORM}"
: > "${LOG_FILE}"
npm run start >>"${LOG_FILE}" 2>&1 &
echo "$!" >"${PID_FILE}"

echo "launch-ui: started pid $(cat "${PID_FILE}")"
echo "launch-ui: waiting for https://localhost:3000"

for _ in $(seq 1 120); do
    if curl -sk --max-time 2 "https://localhost:3000" >/dev/null 2>&1; then
        echo "launch-ui: ready at https://localhost:3000"
        echo "launch-ui: log ${LOG_FILE}"
        exit 0
    fi
    sleep 2
done

echo "launch-ui: timed out; see ${LOG_FILE}"
exit 1
