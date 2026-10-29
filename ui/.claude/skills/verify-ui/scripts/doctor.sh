#!/usr/bin/env bash
# Read-only health check: is this instance worth driving? Exits non-zero on the first hard failure.

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

ok() { echo "  ok    $*"; }
warn() { echo "  warn  $*"; }
fail() { echo "  FAIL  $*"; exit 1; }

echo "verify-ui doctor"

[[ -f "${BACKEND_ENV}" ]] || fail "no backend connected (scripts/backend.sh connect <cluster>)"
load_backend
ok "backend UI_START_TARGET=${UI_START_TARGET}"

if [[ -n "${INFRA_CLUSTER}" ]]; then
    lifespan="$(infractl get "${INFRA_CLUSTER}" 2>/dev/null | awk '/Lifespan:/{print $2}')"
    status="$(infractl get "${INFRA_CLUSTER}" 2>/dev/null | awk '/Status:/{print $2}')"
    [[ "${status}" == "READY" ]] || fail "infra ${INFRA_CLUSTER} status ${status:-unknown}"
    ok "infra ${INFRA_CLUSTER} READY, ${lifespan} left"
fi

code="$(admin_curl "${UI_START_TARGET}/v1/auth/status" -o /dev/null -w '%{http_code}' --max-time 15 || true)"
[[ "${code}" == "200" ]] || fail "Central rejected admin basic auth directly (HTTP ${code:-none})"
ok "Central accepts admin basic auth"

central_version="$(admin_curl "${UI_START_TARGET}/v1/metadata" --max-time 15 | jq -r '.version // "unknown"')"
local_tag="$(local_tag 2>/dev/null || echo unknown)"
if [[ "${central_version}" == "${local_tag}" ]]; then
    ok "Central ${central_version} matches checkout"
else
    warn "Central ${central_version} differs from checkout ${local_tag}; API changes in this branch may 404"
fi

vite_running || fail "dev server not running (scripts/launch.sh)"
pid="$(vite_pid)"
owns=0
pid_owns_port "${pid}" "${VERIFY_UI_PORT}" || owns=$?
case "${owns}" in
0) ok "dev server pid ${pid} owns port ${VERIFY_UI_PORT}" ;;
2) warn "cannot check which pid owns port ${VERIFY_UI_PORT} (no ss or lsof); relying on the app shell check" ;;
*) fail "pid ${pid} is alive but does not own port ${VERIFY_UI_PORT}" ;;
esac

cwd="$(pid_cwd "${pid}")"
if [[ -z "${cwd}" ]]; then
    warn "cannot read the dev server's working directory (no /proc or lsof)"
elif [[ "${cwd}" == "${PLATFORM_DIR}" ]]; then
    ok "dev server serves this checkout ($(git -C "${REPO_DIR}" rev-parse --abbrev-ref HEAD))"
else
    fail "dev server cwd ${cwd} is not this checkout (${PLATFORM_DIR})"
fi

curl -sk "${UI_URL}/" | grep -q 'id="root"' || fail "${UI_URL}/ does not return the app shell"
ok "${UI_URL} returns the app shell"

code="$(admin_curl "${UI_URL}/v1/auth/status" -o /dev/null -w '%{http_code}' --max-time 15 || true)"
[[ "${code}" == "200" ]] || fail "proxied /v1/auth/status through the dev server returned HTTP ${code:-none}"
ok "dev server proxies /v1 to Central"

echo "healthy"
