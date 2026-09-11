#!/usr/bin/env bash
# Read-only health check for ACS console verification.
# Answers: is this instance worth driving?
# Never starts, stops, or mutates Central, the UI, or the cluster.

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${SKILL_DIR}/../../.." && pwd)"

UI_PORT="${VERIFY_ACS_UI_PORT:-3000}"
CENTRAL_BASE="${UI_START_TARGET:-https://localhost:8000}"
PASSWORD_FILE="${ROX_ADMIN_PASSWORD_FILE:-${REPO_ROOT}/deploy/k8s/central-deploy/password}"
EVIDENCE_DIR="${VERIFY_ACS_EVIDENCE_DIR:-/tmp/verify-acs-console}"

PASS=0
FAIL=0
WARN=0
INCONCLUSIVE=0

pass() { echo "PASS  $*"; PASS=$((PASS + 1)); }
fail() { echo "FAIL  $*"; FAIL=$((FAIL + 1)); }
warn() { echo "WARN  $*"; WARN=$((WARN + 1)); }
note() { echo "NOTE  $*"; }
inconclusive() { echo "INCONCLUSIVE  $*"; INCONCLUSIVE=$((INCONCLUSIVE + 1)); }

owning_pid() {
    local port="$1"
    # Prefer the listener PID. ss/lsof may be missing on some machines.
    if command -v ss >/dev/null 2>&1; then
        ss -ltnp 2>/dev/null | awk -v p=":${port}" '$4 ~ p"$" { print; found=1 } END { exit !found }' || true
        return 0
    fi
    if command -v lsof >/dev/null 2>&1; then
        lsof -nP -iTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true
        return 0
    fi
    return 1
}

port_listening() {
    local port="$1"
    if command -v ss >/dev/null 2>&1; then
        ss -ltn 2>/dev/null | awk -v p=":${port}" '$4 ~ p"$" { found=1 } END { exit !found }'
        return $?
    fi
    if command -v lsof >/dev/null 2>&1; then
        lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1
        return $?
    fi
    # Last resort: bash /dev/tcp
    (echo >/dev/tcp/127.0.0.1/"${port}") >/dev/null 2>&1
}

http_code() {
    local url="$1"
    shift
    local code
    code="$(curl -sk -o /dev/null -w '%{http_code}' --max-time 8 "$@" "${url}" 2>/dev/null)" || code="000"
    if [[ -z "${code}" || "${code}" == "000000" ]]; then
        code="000"
    fi
    printf '%s' "${code}"
}

echo "ACS console doctor (read-only)"
echo "repo:    ${REPO_ROOT}"
echo "ui port: ${UI_PORT}"
echo "central: ${CENTRAL_BASE}"
echo "evidence (not created): ${EVIDENCE_DIR}"
echo

# --- Config / deps (can run without Central) ---

if [[ -f "${REPO_ROOT}/ui/README.md" && -f "${REPO_ROOT}/ui/apps/platform/package.json" ]]; then
    pass "ui/README.md and ui/apps/platform/package.json exist"
else
    fail "expected ACS UI tree under ui/ and ui/apps/platform/"
fi

if [[ -f "${REPO_ROOT}/ui/apps/platform/vite.config.js" && -f "${REPO_ROOT}/ui/apps/platform/src/setupProxy.js" ]]; then
    pass "Vite config and setupProxy.js present"
else
    fail "missing Vite/proxy config"
fi

if [[ -f "${REPO_ROOT}/ui/apps/platform/cypress.config.js" && -d "${REPO_ROOT}/ui/apps/platform/cypress/integration" ]]; then
    pass "Cypress e2e harness present (cypress.config.js + cypress/integration)"
else
    fail "Cypress e2e harness missing"
fi

if [[ -f "${REPO_ROOT}/ui/apps/platform/src/routePaths.ts" ]]; then
    pass "routePaths.ts present"
else
    fail "missing ui/apps/platform/src/routePaths.ts"
fi

if command -v node >/dev/null 2>&1; then
    NODE_VER="$(node -v | tr -d 'v')"
    NODE_MAJOR="${NODE_VER%%.*}"
    if [[ "${NODE_MAJOR}" -ge 22 ]]; then
        pass "node ${NODE_VER} (ui engines require >=22.13.0)"
    else
        fail "node ${NODE_VER} is below ui engines >=22.13.0"
    fi
else
    fail "node not on PATH"
fi

if [[ -d "${REPO_ROOT}/ui/apps/platform/node_modules" ]]; then
    pass "ui/apps/platform/node_modules present"
else
    warn "ui/apps/platform/node_modules missing (run npm ci in ui/apps/platform before start)"
fi

if command -v kubectl >/dev/null 2>&1; then
    if kubectl cluster-info >/dev/null 2>&1; then
        warn "kubectl can reach a cluster — treat it as a shared Central/k8s unless this run started it"
        kubectl get ns stackrox >/dev/null 2>&1 && warn "namespace stackrox exists (typical ACS install namespace)"
    else
        note "kubectl present but no reachable cluster"
    fi
else
    note "kubectl not on PATH (local deploy-local cannot run on this machine)"
fi

# --- Live instance ---

UI_UP=0
CENTRAL_UP=0

if port_listening "${UI_PORT}"; then
    UI_UP=1
    pass "port ${UI_PORT} has a listener"
    echo "      owner: $(owning_pid "${UI_PORT}" | tr '\n' ' ')"
    UI_CODE="$(http_code "https://127.0.0.1:${UI_PORT}/login")"
    if [[ "${UI_CODE}" =~ ^(200|301|302|401|403)$ ]]; then
        pass "https://127.0.0.1:${UI_PORT}/login returned HTTP ${UI_CODE}"
    else
        # Vite may use a generated cert; a connection with any HTTP code still counts.
        if [[ "${UI_CODE}" != "000" ]]; then
            pass "https://127.0.0.1:${UI_PORT}/login reachable (HTTP ${UI_CODE})"
        else
            fail "port ${UI_PORT} is listening but HTTPS login is not reachable"
        fi
    fi
else
    fail "nothing listening on UI port ${UI_PORT} (Vite npm run start is down)"
fi

PING_URL="${CENTRAL_BASE%/}/v1/ping"
PING_CODE="$(http_code "${PING_URL}")"
if [[ "${PING_CODE}" == "200" ]]; then
    CENTRAL_UP=1
    pass "Central reachable: ${PING_URL} → HTTP 200"
else
    fail "Central not reachable: ${PING_URL} → HTTP ${PING_CODE} (need live Central; do not substitute Cypress component tests)"
fi

# Auth: password file or env, then /v1/auth/status or token generate is out of scope
# for a read-only check — we only probe an authenticated GET.
ADMIN_USER="${ROX_USERNAME:-admin}"
ADMIN_PASS="${ROX_ADMIN_PASSWORD:-}"
if [[ -z "${ADMIN_PASS}" && -f "${PASSWORD_FILE}" ]]; then
    ADMIN_PASS="$(cat "${PASSWORD_FILE}")"
    note "using admin password from ${PASSWORD_FILE}"
fi

if [[ "${CENTRAL_UP}" -eq 1 ]]; then
    if [[ -n "${ADMIN_PASS}" ]]; then
        AUTH_CODE="$(http_code "${CENTRAL_BASE%/}/v1/auth/status" -u "${ADMIN_USER}:${ADMIN_PASS}")"
        # /v1/auth/status with basic auth typically 200 when the password is valid.
        FEATURE_CODE="$(http_code "${CENTRAL_BASE%/}/v1/featureflags" -u "${ADMIN_USER}:${ADMIN_PASS}")"
        if [[ "${AUTH_CODE}" == "200" || "${FEATURE_CODE}" == "200" ]]; then
            pass "admin basic auth accepted (auth/status=${AUTH_CODE} featureflags=${FEATURE_CODE})"
        else
            fail "admin basic auth rejected (auth/status=${AUTH_CODE} featureflags=${FEATURE_CODE})"
        fi
    else
        fail "Central is up but no ROX_ADMIN_PASSWORD and no ${PASSWORD_FILE}"
    fi
else
    if [[ -f "${PASSWORD_FILE}" ]]; then
        note "password file exists but Central is down — auth not probed"
    else
        note "no password file at ${PASSWORD_FILE} (expected after npm run deploy-local)"
    fi
fi

echo
echo "summary: pass=${PASS} fail=${FAIL} warn=${WARN}"

if [[ "${CENTRAL_UP}" -eq 0 ]]; then
    echo
    echo "VERDICT: INCONCLUSIVE"
    echo "Central is not up. ACS console proof requires a live Central (and usually k8s)."
    echo "Do not treat Cypress component tests, Vitest, or mocked GraphQL as console proof."
    echo "Do not invent a passing live run."
    exit 2
fi

if [[ "${UI_UP}" -eq 0 || "${FAIL}" -gt 0 ]]; then
    echo
    echo "VERDICT: NOT READY"
    echo "Fix the FAIL lines before driving the console."
    exit 1
fi

echo
echo "VERDICT: READY"
echo "Drive with Cypress e2e against this Central, or browser/CDP on https://localhost:${UI_PORT}."
exit 0
