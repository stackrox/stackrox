#!/usr/bin/env bash
# Run one proof spec headless against the running dev server and collect evidence.
#
#   drive.sh <spec-path>      spec from new-spec.sh (or any spec, e.g. cypress/integration/...)
#
# Evidence lands in <run>/evidence/: screenshots, video, *.state.json, console-errors.json,
# checks.json, cypress.log, summary.txt. The run dir is printed on the last line.
#
# Proof gate: even when Cypress passes, the proof fails if any test ran no assertions in its
# body, or if checks.json is missing (the evidence support file did not run).

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

spec="$(cd "$(dirname "${1:?usage: drive.sh <spec-path>}")" && pwd)/$(basename "$1")"
[[ -f "${spec}" ]] || die "no spec at ${spec}"

load_backend
vite_running || die "dev server not running; run scripts/launch.sh"

case "${spec}" in
"${RUNS_DIR}"/*/specs/*) run_dir="${spec%/specs/*}" ;;
*) run_dir="${RUNS_DIR}/$(date +%Y%m%d-%H%M%S)-$(basename "${spec}" | cut -d. -f1)" ;;
esac
evidence="${run_dir}/evidence"
rm -rf "${evidence}"
mkdir -p "${evidence}"
run_id="$(basename "${run_dir}")"

# Per-run Admin API token, named so cleanup.sh can revoke it. It also expires on its own
# (VERIFY_UI_TOKEN_TTL_HOURS, default 4) in case cleanup never runs.
token_body="$(jq -cn --arg name "verify-ui-${run_id}" --argjson hours "${VERIFY_UI_TOKEN_TTL_HOURS:-4}" \
    '{name: $name, roles: ["Admin"], expiration: (now + $hours * 3600 | floor | todate)}')"
token="$(admin_curl "${UI_URL}/v1/apitokens/generate" -X POST -d "${token_body}" | jq -r '.token // ""')"
[[ -n "${token}" ]] || die "Central did not issue an API token; run scripts/doctor.sh"
export CYPRESS_ROX_AUTH_TOKEN="${token}"

# Mirror scripts/cypress.sh: expose Central feature flags so helpers/features.js sees them.
while IFS=$'\t' read -r name enabled; do
    export "CYPRESS_${name}=${enabled}"
done < <(admin_curl "${UI_URL}/v1/featureflags" | jq -r '.featureFlags[] | [.envVar, .enabled] | @tsv')

export CYPRESS_VERIFY_EVIDENCE_DIR="${evidence}"
export CYPRESS_CRASH_REPORTS=0
export TZ=UTC
# Skill config: repo config plus a window big enough for the full viewport.
export VERIFY_PLATFORM_DIR="${PLATFORM_DIR}"

cp "${spec}" "${evidence}/"
set +e
(cd "${PLATFORM_DIR}" && npx cypress run --e2e --browser electron --config-file "${SKILL_DIR}/templates/cypress.config.js" --spec "${spec}" --config \
    "baseUrl=${UI_URL},specPattern=$(dirname "${spec}")/*.{js,ts},screenshotsFolder=${evidence}/screenshots,videosFolder=${evidence}/videos,retries=0,video=true,trashAssetsBeforeRuns=false") \
    > "${evidence}/cypress.log" 2>&1
status=$?
set -e

gate=()
checks="${evidence}/checks.json"
if [[ ! -f "${checks}" ]]; then
    gate+=("no checks.json, so the evidence support file did not run; see cypress.log")
elif [[ "$(jq '.assertions | length' "${checks}")" -eq 0 ]]; then
    gate+=("no tests ran")
else
    while IFS= read -r test; do
        gate+=("\"${test}\" ran no assertions; check the visible result with should() or expect()")
    done < <(jq -r '.assertions | to_entries[] | select(.value == 0) | .key' "${checks}")
fi

if [[ ${status} -ne 0 ]]; then
    result="FAIL (exit ${status})"
elif [[ ${#gate[@]} -gt 0 ]]; then
    result="FAIL (proof gate)"
    status=1
else
    result="PASS"
fi

{
    echo "spec: ${spec}"
    echo "result: ${result}"
    echo "checkout: $(git -C "${REPO_DIR}" rev-parse --abbrev-ref HEAD) @ $(git -C "${REPO_DIR}" rev-parse --short HEAD)$(git -C "${REPO_DIR}" diff --quiet || echo ' + uncommitted changes')"
    echo "central: ${UI_START_TARGET} version $(admin_curl "${UI_START_TARGET}/v1/metadata" | jq -r '.version // "unknown"')"
    echo "time: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    if [[ -f "${checks}" ]]; then jq -r '.assertions | to_entries[] | "assertions: \(.value) in \"\(.key)\""' "${checks}"; fi
    for line in ${gate[@]+"${gate[@]}"}; do echo "proof gate: ${line}"; done
} > "${evidence}/summary.txt"

sed -n '/(Run Finished)/,$p' "${evidence}/cypress.log" | tail -8 >&2
[[ ${status} -eq 0 ]] || grep -E 'AssertionError|CypressError|Error:' "${evidence}/cypress.log" | head -5 >&2 || true
for line in ${gate[@]+"${gate[@]}"}; do info "proof gate: ${line}"; done
echo "${run_dir}"
exit "${status}"
