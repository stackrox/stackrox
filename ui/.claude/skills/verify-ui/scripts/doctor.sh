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
[[ "${code}" == "200" ]] || fail "Central at ${UI_START_TARGET} $(describe_http_failure "${code}")"
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
ok "dev server pid ${pid} owns port ${VERIFY_UI_PORT} and serves this checkout"

curl -sk "${UI_URL}/" | grep -q 'id="root"' || fail "${UI_URL}/ does not return the app shell"
ok "${UI_URL} returns the app shell"

code="$(admin_curl "${UI_URL}/v1/auth/status" -o /dev/null -w '%{http_code}' --max-time 15 || true)"
[[ "${code}" == "200" ]] || fail "Central through the dev server proxy ${UI_URL}/v1 $(describe_http_failure "${code}")"
ok "dev server proxies /v1 to Central"

echo "healthy"
