#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PLATFORM="${ROOT}/ui/apps/platform"

failures=0
check() {
    local label="$1"
    shift
    if "$@"; then
        echo "ok  ${label}"
    else
        echo "FAIL ${label}"
        failures=$((failures + 1))
    fi
}

echo "verify-acs-console doctor (read-only)"
echo "repo root: ${ROOT}"

check "ui/apps/platform exists" test -d "${PLATFORM}"
check "package.json present" test -f "${PLATFORM}/package.json"
check "node_modules installed" test -d "${PLATFORM}/node_modules"
check "cypress.config.js present" test -f "${PLATFORM}/cypress.config.js"

if command -v node >/dev/null 2>&1; then
    node_major="$(node -p "process.versions.node.split('.')[0]")"
    node_minor="$(node -p "process.versions.node.split('.')[1]")"
    if [[ "${node_major}" -ge 22 ]] && [[ "${node_minor}" -ge 13 || "${node_major}" -gt 22 ]]; then
        echo "ok  node >= 22.13.0 ($(node -v))"
    else
        echo "FAIL node >= 22.13.0 required (have $(node -v))"
        failures=$((failures + 1))
    fi
else
    echo "FAIL node not found"
    failures=$((failures + 1))
fi

if [[ -x "${PLATFORM}/node_modules/.bin/cypress" ]]; then
    echo "ok  cypress binary"
else
    echo "FAIL cypress binary missing (run: cd ui/apps/platform && npm ci)"
    failures=$((failures + 1))
fi

if curl -sk --max-time 3 "https://localhost:3000" >/dev/null 2>&1; then
    echo "ok  dev UI responding at https://localhost:3000"
else
    echo "info dev UI not running at https://localhost:3000 (expected unless launch-ui.sh is active)"
fi

if curl -sk --max-time 3 "https://localhost:8000/v1/meta/features" >/dev/null 2>&1; then
    echo "ok  Central API reachable at https://localhost:8000"
else
    echo "info Central not reachable at https://localhost:8000 (required for E2E, not for component tests)"
fi

if [[ "${failures}" -gt 0 ]]; then
    echo "doctor: ${failures} check(s) failed"
    exit 1
fi

echo "doctor: ready"
