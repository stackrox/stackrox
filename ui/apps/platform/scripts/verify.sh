#!/usr/bin/env bash

# Verification harness for UI changes: prove a change works in the running app.
# Agents follow .claude/skills/verify-ui/SKILL.md at the repo root; people can run it directly.
#
# Usage:
#   scripts/verify.sh doctor              check the dev server, Central, and auth are ready
#   scripts/verify.sh static [files...]   tsc, then eslint and vitest on changed files
#   scripts/verify.sh open <route>        open a route and capture evidence
#   scripts/verify.sh prove [feature]     run the e2e specs for a feature (lists features if omitted)
#
# Environment:
#   UI_BASE_URL          dev server URL (default https://localhost:3000)
#   ROX_AUTH_TOKEN       API token to reuse; otherwise one is minted from basic auth creds
#   ROX_USERNAME, ROX_ADMIN_PASSWORD
#                        basic auth creds; otherwise read from deploy/k8s/central-deploy/password
#   VERIFY_BASE_REF      ref that changed files are compared against (default origin/master)
#   VERIFY_EVIDENCE_DIR  where evidence goes (default ui/apps/platform/verify-evidence)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${PLATFORM_DIR}/../../.." && pwd)"

UI_BASE_URL="${UI_BASE_URL:-https://localhost:3000}"
VERIFY_BASE_REF="${VERIFY_BASE_REF:-origin/master}"
EVIDENCE_ROOT="${VERIFY_EVIDENCE_DIR:-${PLATFORM_DIR}/verify-evidence}"
PASSWORD_FILE="${REPO_ROOT}/deploy/k8s/central-deploy/password"

cd "${PLATFORM_DIR}"

pass() { echo "  PASS  $*"; }
fail() { echo "  FAIL  $*"; }
info() { echo "  INFO  $*"; }
die() {
    echo >&2 "ERROR: $*"
    exit 1
}

usage() {
    sed -n '6,10p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
    exit 2
}

# Use built-in echo to not expose the value in the process list.
curl_cfg() {
    echo -n "$1 = \"${2//[\"\\]/\\&}\""
}

load_basic_auth_creds() {
    if [[ -n "${ROX_USERNAME:-}" && -n "${ROX_ADMIN_PASSWORD:-}" ]]; then
        return 0
    fi
    if [[ -f "${PASSWORD_FILE}" ]]; then
        ROX_USERNAME='admin'
        ROX_ADMIN_PASSWORD="$(cat "${PASSWORD_FILE}")"
        return 0
    fi
    return 1
}

# Calls the API through the dev server proxy, with the token if set, else basic auth.
api_get() {
    local path="$1"
    if [[ -n "${ROX_AUTH_TOKEN:-}" ]]; then
        curl -sk --max-time 10 --config <(curl_cfg header "Authorization: Bearer ${ROX_AUTH_TOKEN}") \
            "${UI_BASE_URL}${path}"
    else
        curl -sk --max-time 10 --config <(curl_cfg user "${ROX_USERNAME}:${ROX_ADMIN_PASSWORD}") \
            "${UI_BASE_URL}${path}"
    fi
}

http_status() {
    curl -sk --max-time 10 -o /dev/null -w '%{http_code}' "$1" || true
}

ensure_auth_token() {
    if [[ -n "${ROX_AUTH_TOKEN:-}" ]]; then
        return 0
    fi
    load_basic_auth_creds || die "No ROX_AUTH_TOKEN and no basic auth creds. Run: scripts/verify.sh doctor"
    ROX_AUTH_TOKEN="$(UI_BASE_URL="${UI_BASE_URL}" ROX_USERNAME="${ROX_USERNAME}" \
        ROX_ADMIN_PASSWORD="${ROX_ADMIN_PASSWORD}" UI_API_TOKEN_NAME=ui_verify "${SCRIPT_DIR}/get-auth-token.sh")" ||
        die "Could not mint an auth token. Run: scripts/verify.sh doctor"
    echo "Minted a new API token. Export ROX_AUTH_TOKEN to reuse one across runs."
}

new_evidence_dir() {
    local label="$1"
    local slug
    slug="$(echo "${label}" | tr -c 'A-Za-z0-9' '-' | sed -e 's/--*/-/g' -e 's/^-//' -e 's/-$//')"
    local dir
    dir="${EVIDENCE_ROOT}/$(date +%Y%m%d-%H%M%S)-${slug:-root}"
    mkdir -p "${dir}"
    echo "${dir}"
}

cmd_doctor() {
    local failures=0

    echo "Checking the verification environment (UI_BASE_URL=${UI_BASE_URL})"

    for tool in curl jq npx; do
        if command -v "${tool}" >/dev/null; then
            pass "${tool} is installed"
        else
            fail "${tool} is not installed"
            failures=$((failures + 1))
        fi
    done

    if [[ -d node_modules/.bin ]]; then
        pass "node_modules is installed"
    else
        fail "node_modules is missing. Run: npm ci (from ui/)"
        failures=$((failures + 1))
    fi

    local status
    status="$(http_status "${UI_BASE_URL}/")"
    if [[ "${status}" == "200" ]]; then
        pass "dev server responds at ${UI_BASE_URL}"
    else
        fail "dev server did not respond at ${UI_BASE_URL} (HTTP ${status}). Start it in the background with: BROWSER=none npm run start"
        failures=$((failures + 1))
    fi

    status="$(http_status "${UI_BASE_URL}/v1/ping")"
    if [[ "${status}" == "200" ]]; then
        pass "Central responds through the dev server proxy"
    else
        fail "Central did not respond through the proxy (HTTP ${status}). Check that Central is running and UI_START_TARGET points at it (default https://localhost:8000)"
        failures=$((failures + 1))
        echo "Fix the failures above, then run doctor again."
        return 1
    fi

    if [[ -n "${ROX_AUTH_TOKEN:-}" ]]; then
        info "using ROX_AUTH_TOKEN from the environment"
    elif load_basic_auth_creds; then
        info "using basic auth creds for ${ROX_USERNAME}; open and prove mint a token from them"
    else
        fail "no ROX_AUTH_TOKEN, no ROX_USERNAME and ROX_ADMIN_PASSWORD, and no ${PASSWORD_FILE}"
        failures=$((failures + 1))
        echo "Fix the failures above, then run doctor again."
        return 1
    fi

    local auth_status
    auth_status="$(api_get /v1/auth/status | jq -r '.userId // empty' 2>/dev/null || true)"
    if [[ -n "${auth_status}" ]]; then
        pass "authenticated as ${auth_status}"
    else
        fail "credentials were rejected by /v1/auth/status"
        failures=$((failures + 1))
    fi

    local flags
    flags="$(api_get /v1/featureflags | jq -r '.featureFlags[]? | select(.enabled) | .envVar' 2>/dev/null || true)"
    if [[ -n "${flags}" ]]; then
        info "enabled feature flags:"
        while IFS= read -r flag; do
            echo "          ${flag}"
        done <<<"${flags}"
    else
        info "no enabled feature flags reported"
    fi

    if [[ "${failures}" -ne 0 ]]; then
        echo "Fix the failures above, then run doctor again."
        return 1
    fi
    echo "Ready to verify."
}

# Prints changed JS and TS files under ui/apps/platform, relative to it.
changed_files() {
    local merge_base
    merge_base="$(git merge-base "${VERIFY_BASE_REF}" HEAD)" ||
        die "Could not find the merge base with ${VERIFY_BASE_REF}. Set VERIFY_BASE_REF."
    {
        git diff --name-only --relative --diff-filter=ACMR "${merge_base}" -- .
        git ls-files --others --exclude-standard
    } | grep -E '\.(js|jsx|ts|tsx)$' | sort -u | while read -r file; do
        [[ -f "${file}" ]] && echo "${file}"
    done
}

cmd_static() {
    local files=()
    if [[ $# -gt 0 ]]; then
        files=("$@")
    else
        while IFS= read -r file; do
            files+=("${file}")
        done < <(changed_files)
    fi

    local src_files=()
    for file in "${files[@]+"${files[@]}"}"; do
        [[ "${file}" == src/* ]] && src_files+=("${file}")
    done

    local failures=()

    echo "==> tsc"
    npm run --silent tsc || failures+=("tsc")

    if [[ ${#files[@]} -eq 0 ]]; then
        echo "==> eslint and vitest skipped: no changed JS or TS files"
    else
        echo "==> eslint on ${#files[@]} file(s)"
        npx eslint --quiet --cache --cache-strategy content "${files[@]}" || failures+=("eslint")

        if [[ ${#src_files[@]} -eq 0 ]]; then
            echo "==> vitest skipped: no changed files under src/"
        else
            echo "==> vitest related to ${#src_files[@]} file(s)"
            TZ=UTC npx vitest related --run --passWithNoTests "${src_files[@]}" || failures+=("vitest")
        fi
    fi

    if [[ ${#failures[@]} -ne 0 ]]; then
        echo "static: FAIL (${failures[*]})"
        return 1
    fi
    echo "static: PASS"
}

cmd_open() {
    local route="${1:-}"
    [[ "${route}" == /* ]] || die "Usage: scripts/verify.sh open <route>, for example /main/violations"

    ensure_auth_token
    local dir
    dir="$(new_evidence_dir "${route}")"

    echo "Opening ${route}. Evidence goes to ${dir}"
    local result=0
    CYPRESS_ROX_AUTH_TOKEN="${ROX_AUTH_TOKEN}" \
        CYPRESS_VERIFY_ROUTE="${route}" \
        CYPRESS_VERIFY_EVIDENCE_DIR="${dir}" \
        TZ=UTC npx cypress run --e2e --browser "${VERIFY_BROWSER:-electron}" \
        --spec cypress/verify/openRoute.test.js \
        --config "baseUrl=${UI_BASE_URL},specPattern=cypress/verify/*.test.js,video=false,retries=0,screenshotsFolder=${dir}" ||
        result=$?

    local report="${dir}/report.json"
    if [[ -f "${report}" ]]; then
        echo
        jq -r '"route:     \(.route)",
            "final url: \(.finalUrl)",
            "heading:   \(.heading)",
            "failures:  \(if (.failures | length) == 0 then "none" else (.failures | join("; ")) end)",
            "warnings:  \(.warnings.consoleErrors | length) console error(s), \(.warnings.a11yViolations | length) a11y violation(s)"' \
            "${report}"
        echo "report:     ${report}"
        find "${dir}" -name '*.png' -print | sed 's/^/screenshot: /'
    else
        echo "No report was written. The run failed before capturing evidence; run doctor again."
    fi
    return "${result}"
}

list_features() {
    echo "Features (directories under cypress/integration):"
    find cypress/integration -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort | sed 's/^/  /'
}

cmd_prove() {
    local feature="${1:-}"
    if [[ -z "${feature}" ]]; then
        list_features
        return 0
    fi

    local spec
    if [[ -d "cypress/integration/${feature}" ]]; then
        spec="${feature}/**/*.test.*"
    elif [[ -f "cypress/integration/${feature}" ]]; then
        spec="${feature}"
    else
        echo "Unknown feature: ${feature}"
        list_features
        return 1
    fi

    # cypress.sh mints its own token from basic auth creds.
    load_basic_auth_creds || die "No basic auth creds for cypress.sh. Run: scripts/verify.sh doctor"
    local dir
    dir="$(new_evidence_dir "prove-${feature}")"
    echo "Running e2e specs for ${feature}. Evidence goes to ${dir}"

    # cypress.sh uses UI_BASE_URL for both the API and the app, which works through the dev server proxy.
    # It calls a bare `cypress`, which npm scripts find through node_modules/.bin, so add that to PATH here.
    PATH="${PLATFORM_DIR}/node_modules/.bin:${PATH}" \
        UI_BASE_URL="${UI_BASE_URL}" ROX_USERNAME="${ROX_USERNAME}" ROX_ADMIN_PASSWORD="${ROX_ADMIN_PASSWORD}" \
        TEST_RESULTS_OUTPUT_DIR="${dir}" TZ=UTC \
        "${SCRIPT_DIR}/cypress.sh" run --spec "${spec}"
}

main() {
    local command="${1:-}"
    [[ $# -gt 0 ]] && shift
    case "${command}" in
        doctor) cmd_doctor "$@" ;;
        static) cmd_static "$@" ;;
        open) cmd_open "$@" ;;
        prove) cmd_prove "$@" ;;
        *) usage ;;
    esac
}

main "$@"
