#!/usr/bin/env bash
# Shared paths and helpers for verify-ui scripts. Source it; do not run it.

set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UI_DIR="$(cd "${SKILL_DIR}/../../.." && pwd)"
PLATFORM_DIR="${UI_DIR}/apps/platform"
REPO_DIR="$(cd "${UI_DIR}/.." && pwd)"

VERIFY_UI_HOME="${VERIFY_UI_HOME:-${HOME}/.cache/stackrox-verify-ui}"
STATE_DIR="${VERIFY_UI_HOME}/state"
RUNS_DIR="${VERIFY_UI_HOME}/runs"
BACKEND_ENV="${STATE_DIR}/backend.env"
PASSWORD_FILE="${STATE_DIR}/admin-password"
VITE_PID_FILE="${STATE_DIR}/vite.pid"
VITE_LOG="${STATE_DIR}/vite.log"
VERIFY_UI_PORT="${VERIFY_UI_PORT:-3000}"
UI_URL="https://localhost:${VERIFY_UI_PORT}"

mkdir -p "${STATE_DIR}" "${RUNS_DIR}"
chmod 700 "${VERIFY_UI_HOME}" "${STATE_DIR}"

die() { echo "verify-ui: $*" >&2; exit 1; }
info() { echo "verify-ui: $*" >&2; }

# Writes a curl config line so secrets never appear in the process list.
curl_cfg() { echo -n "$1 = \"${2//[\"\\]/\\&}\""; }

load_backend() {
    [[ -f "${BACKEND_ENV}" ]] || die "no backend connected; run scripts/backend.sh connect <cluster> first"
    # shellcheck disable=SC1090
    source "${BACKEND_ENV}"
    [[ -f "${PASSWORD_FILE}" ]] || die "missing ${PASSWORD_FILE}; rerun scripts/backend.sh connect"
}

# curl against Central (or the UI proxy) with admin basic auth.
admin_curl() {
    local url="$1"; shift
    curl -sk --config <(curl_cfg user "admin:$(cat "${PASSWORD_FILE}")") "$@" "${url}"
}

vite_pid() {
    [[ -f "${VITE_PID_FILE}" ]] && cat "${VITE_PID_FILE}" || true
}

vite_running() {
    local pid
    pid="$(vite_pid)"
    [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null
}
