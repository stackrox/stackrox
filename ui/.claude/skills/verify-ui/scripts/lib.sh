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
TOKENS_DIR="${STATE_DIR}/tokens"
ACTIVE_RUNS_DIR="${STATE_DIR}/active-runs"
LIFECYCLE_LOCK="${STATE_DIR}/lifecycle.lock"
VERIFY_UI_PORT="${VERIFY_UI_PORT:-3000}"
UI_URL="https://localhost:${VERIFY_UI_PORT}"

mkdir -p "${STATE_DIR}" "${RUNS_DIR}" "${TOKENS_DIR}" "${ACTIVE_RUNS_DIR}"
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

# curl with basic auth for user $1 and the password in file $2.
basic_curl() {
    local user="$1" password_file="$2" url="$3"; shift 3
    curl -sk --config <(curl_cfg user "${user}:$(cat "${password_file}")") "$@" "${url}"
}

# curl against Central (or the UI proxy) with the connected backend's admin basic auth.
admin_curl() {
    local url="$1"; shift
    basic_curl "${CENTRAL_USER:-admin}" "${PASSWORD_FILE}" "${url}" "$@"
}

# Explain a failed HTTP status from Central in words.
describe_http_failure() {
    case "${1:-000}" in
    000) echo "is unreachable (no HTTP response)" ;;
    401 | 403) echo "rejected the admin credentials (HTTP $1)" ;;
    *) echo "answered HTTP $1 to /v1/auth/status" ;;
    esac
}

vite_pid() {
    [[ -f "${VITE_PID_FILE}" ]] && cat "${VITE_PID_FILE}" || true
}

vite_pid_alive() {
    local pid
    pid="$(vite_pid)"
    [[ "${pid}" =~ ^[0-9]+$ ]] && kill -0 "${pid}" 2>/dev/null
}

# A live recorded pid is safe to reuse or stop only if it owns this port and checkout.
vite_running() {
    local pid cwd owns=0
    pid="$(vite_pid)"
    [[ -n "${pid}" ]] || return 1
    [[ "${pid}" =~ ^[0-9]+$ ]] || die "invalid dev server pid in ${VITE_PID_FILE}; inspect and remove that file"
    kill -0 "${pid}" 2>/dev/null || return 1
    pid_owns_port "${pid}" "${VERIFY_UI_PORT}" || owns=$?
    [[ "${owns}" -eq 0 ]] || die "cannot verify that dev server pid ${pid} owns port ${VERIFY_UI_PORT}; inspect ${VITE_PID_FILE} and the port before retrying"
    cwd="$(pid_cwd "${pid}")"
    [[ "${cwd}" == "${PLATFORM_DIR}" ]] || die "cannot verify that dev server pid ${pid} runs from ${PLATFORM_DIR} (cwd: ${cwd:-unknown}); inspect ${VITE_PID_FILE} before retrying"
}

acquire_lifecycle_lock() {
    local attempt
    for attempt in $(seq 1 100); do
        if mkdir "${LIFECYCLE_LOCK}" 2>/dev/null; then
            return 0
        fi
        sleep 0.1
    done
    die "state lock ${LIFECYCLE_LOCK} is busy; retry, or remove it if no verify-ui script is running"
}

release_lifecycle_lock() {
    rmdir "${LIFECYCLE_LOCK}"
}

process_start() {
    local start
    start="$(ps -p "$1" -o lstart= 2>/dev/null | sed 's/^[[:space:]]*//; s/[[:space:]]*$//' || true)"
    if [[ -n "${start}" ]]; then
        printf '%s\n' "${start}"
    elif kill -0 "$1" 2>/dev/null; then
        # Restricted shells can deny ps. A reused pid then causes a safe cleanup refusal.
        printf 'pid:%s\n' "$1"
    fi
}

process_matches_start() {
    local pid="$1" start="$2"
    [[ "${pid}" =~ ^[0-9]+$ && -n "${start}" ]] || return 1
    if [[ "${start}" == "pid:${pid}" ]]; then
        kill -0 "${pid}" 2>/dev/null
    else
        [[ "$(process_start "${pid}")" == "${start}" ]]
    fi
}

active_runs() {
    local run pid start
    for run in "${ACTIVE_RUNS_DIR}"/*; do
        [[ -d "${run}" ]] || continue
        pid="$(cat "${run}/pid" 2>/dev/null || true)"
        start="$(cat "${run}/start" 2>/dev/null || true)"
        if process_matches_start "${pid}" "${start}"; then
            printf '%s (%s)\n' "$(basename "${run}")" "$(cat "${run}/spec" 2>/dev/null || echo unknown-spec)"
        else
            rm -rf "${run}"
        fi
    done
}

revoke_recorded_token() {
    local record="$1" id target
    [[ -f "${record}/id" && -f "${record}/target" ]] || return 0
    id="$(cat "${record}/id")"
    target="$(cat "${record}/target")"
    [[ "${target}" == "${UI_START_TARGET}" ]] || return 1
    if admin_curl "${target}/v1/apitokens/revoke/${id}" -X PATCH -f -o /dev/null --max-time 15; then
        rm -rf "${record}"
    else
        return 1
    fi
}

# Version tag of this checkout, as `make tag` prints it. Falls back to the same git describe
# call from make/env.mk when make is not installed.
local_tag() {
    if command -v make >/dev/null 2>&1; then
        make -C "${REPO_DIR}" --no-print-directory tag
    else
        git -C "${REPO_DIR}" describe --tags --abbrev=10 --dirty --long --exclude '*-nightly-*'
    fi
}

# True when anything listens on TCP port $1 on this host. Uses ss, then lsof, then a connect.
port_in_use() {
    local port="$1"
    if command -v ss >/dev/null 2>&1; then
        [[ -n "$(ss -Hltn "sport = :${port}" 2>/dev/null)" ]]
    elif command -v lsof >/dev/null 2>&1; then
        lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    else
        (exec 3<>"/dev/tcp/127.0.0.1/${port}") 2>/dev/null || (exec 3<>"/dev/tcp/::1/${port}") 2>/dev/null
    fi
}

# Whether pid $1 listens on TCP port $2: returns 0 yes, 1 no, 2 cannot tell (no ss or lsof).
pid_owns_port() {
    local pid="$1" port="$2"
    if command -v ss >/dev/null 2>&1; then
        ss -Hltnp "sport = :${port}" 2>/dev/null | grep -q "pid=${pid},"
    elif command -v lsof >/dev/null 2>&1; then
        lsof -nP -a -p "${pid}" -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
    else
        return 2
    fi
}

# Working directory of pid $1, from /proc on Linux or lsof elsewhere. Empty when unknown.
pid_cwd() {
    local pid="$1"
    if [[ -e "/proc/${pid}/cwd" ]]; then
        readlink "/proc/${pid}/cwd" || true
    elif command -v lsof >/dev/null 2>&1; then
        lsof -a -p "${pid}" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p'
    fi
}
