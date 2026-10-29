#!/usr/bin/env bash
# Run one proof spec headless against the running dev server and collect evidence.
#
#   drive.sh <spec-path>      spec from new-spec.sh (or any spec, e.g. cypress/integration/...)
#
# Each run gets a new <run>/attempt-N/ (N counts up, <run>/latest points at the newest, and
# only the last VERIFY_UI_KEEP_ATTEMPTS, default 5, are kept). It holds screenshots, video,
# *.state.json, errors.txt, checks.json, cypress.log, summary.txt. The attempt dir is printed
# on the last line.
#
# Proof gate: even when Cypress passes, the proof fails if any test ran no assertions in its
# body, if the app threw an uncaught exception that is not known benign, or if checks.json
# is missing (the evidence support file did not run). console.error calls are recorded in
# errors.txt but do not fail the proof.

source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

spec="$(cd "$(dirname "${1:?usage: drive.sh <spec-path>}")" && pwd)/$(basename "$1")"
[[ -f "${spec}" ]] || die "no spec at ${spec}"

load_backend
lock_held=0
active_run_dir=""
token_record=""
run_dir=""
finish_run() {
    local status=$?
    trap - EXIT
    if [[ -n "${token_record}" && -f "${token_record}/id" ]]; then
        revoke_recorded_token "${token_record}" || info "could not revoke this run's token; retry scripts/cleanup.sh"
    elif [[ -n "${token_record}" ]]; then
        rm -rf "${token_record}"
    fi
    if [[ -n "${active_run_dir}" ]]; then
        if [[ "${lock_held}" -eq 0 ]]; then
            acquire_lifecycle_lock
            lock_held=1
        fi
        rm -rf "${active_run_dir}"
        if [[ -n "${run_dir}" ]] && declare -F prune_attempts >/dev/null; then
            prune_attempts
        fi
    fi
    if [[ "${lock_held}" -eq 1 ]]; then
        release_lifecycle_lock
    fi
    exit "${status}"
}

acquire_lifecycle_lock
lock_held=1
trap finish_run EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
active_run_dir="$(mktemp -d "${ACTIVE_RUNS_DIR}/drive.XXXXXX")"
printf '%s\n' "$$" > "${active_run_dir}/pid"
printf '%s\n' "${spec}" > "${active_run_dir}/spec"
process_start "$$" > "${active_run_dir}/start"
[[ -s "${active_run_dir}/start" ]] || die "cannot identify this run's process; cleanup would be unsafe"
vite_running || die "dev server not running; run scripts/launch.sh"

case "${spec}" in
"${RUNS_DIR}"/*/specs/*) run_dir="${spec%/specs/*}" ;;
*) run_dir="${RUNS_DIR}/$(basename "${spec}" | cut -d. -f1)" ;;
esac
mkdir -p "${run_dir}"
keep="${VERIFY_UI_KEEP_ATTEMPTS:-5}"
[[ "${keep}" =~ ^[1-9][0-9]*$ ]] || die "VERIFY_UI_KEEP_ATTEMPTS must be a whole number of 1 or more"

# Attempt numbers already in run_dir, oldest first.
attempts() {
    local dir
    for dir in "${run_dir}"/attempt-*; do
        if [[ "${dir##*/attempt-}" =~ ^[0-9]+$ ]]; then echo "${dir##*/attempt-}"; fi
    done | sort -n
}

attempt_active() {
    local active
    for active in "${ACTIVE_RUNS_DIR}"/*; do
        [[ -f "${active}/evidence" ]] || continue
        [[ "$(cat "${active}/evidence")" == "$1" ]] && return 0
    done
    return 1
}

prune_attempts() {
    local total old latest
    active_runs >/dev/null
    total="$(attempts | wc -l | tr -d ' ')"
    latest="$(readlink "${run_dir}/latest" 2>/dev/null || true)"
    for old in $(attempts); do
        [[ "${total}" -gt "${keep}" ]] || break
        if [[ "attempt-${old}" != "${latest}" ]] && ! attempt_active "${run_dir}/attempt-${old}"; then
            rm -rf "${run_dir:?}/attempt-${old}"
            total=$((total - 1))
        fi
    done
}

# A rerun never overwrites earlier evidence. mkdir without -p claims the number, so two
# runs of one spec at once still get separate attempts.
n=$(( $(attempts | tail -1) + 1 ))
until mkdir "${run_dir}/attempt-${n}" 2>/dev/null; do
    [[ -e "${run_dir}/attempt-${n}" ]] || die "cannot create ${run_dir}/attempt-${n}"
    n=$((n + 1))
done
evidence="${run_dir}/attempt-${n}"
printf '%s\n' "${evidence}" > "${active_run_dir}/evidence"
ln -sfn "attempt-${n}" "${run_dir}/latest"
run_id="$(basename "${run_dir}")-attempt-${n}"
release_lifecycle_lock
lock_held=0

# The token ID is recorded outside evidence so cleanup can revoke only this home's tokens.
token_record="$(mktemp -d "${TOKENS_DIR}/token.XXXXXX")"
printf '%s\n' "${UI_START_TARGET}" > "${token_record}/target"
token_body="$(jq -cn --arg name "verify-ui-${run_id}" --argjson hours "${VERIFY_UI_TOKEN_TTL_HOURS:-4}" \
    '{name: $name, roles: ["Admin"], expiration: (now + $hours * 3600 | floor | todate)}')"
token_response="$(admin_curl "${UI_URL}/v1/apitokens/generate" -X POST -f -d "${token_body}")"
token="$(jq -r '.token // empty' <<< "${token_response}")"
token_id="$(jq -r '.metadata.id // empty' <<< "${token_response}")"
[[ -n "${token}" && -n "${token_id}" ]] || die "Central did not issue a trackable API token; run scripts/doctor.sh"
printf '%s\n' "${token_id}" > "${token_record}/id"
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
else
    [[ "$(jq '.assertions | length' "${checks}")" -gt 0 ]] || gate+=("no tests ran")
    while IFS= read -r test; do
        gate+=("\"${test}\" ran no assertions; check the visible result with should() or expect()")
    done < <(jq -r '.assertions | to_entries[] | select(.value == 0) | .key' "${checks}")
    jq -r '.errors[] | "\(.kind): \(.test): \(.message | gsub("\\s+"; " ") | .[0:500])"' "${checks}" > "${evidence}/errors.txt"
    uncaught="$(jq '[.errors[] | select(.kind == "uncaught")] | length' "${checks}")"
    [[ ${uncaught} -eq 0 ]] || gate+=("the app threw ${uncaught} uncaught exception(s); see errors.txt")
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
    if [[ -f "${checks}" ]]; then
        jq -r '.assertions | to_entries[] | "assertions: \(.value) in \"\(.key)\""' "${checks}"
        jq -r '.errors | group_by(.kind)[] | "errors: \(length) \(.[0].kind)"' "${checks}"
    fi
    for line in ${gate[@]+"${gate[@]}"}; do echo "proof gate: ${line}"; done
} > "${evidence}/summary.txt"

sed -n '/(Run Finished)/,$p' "${evidence}/cypress.log" | tail -8 >&2
[[ ${status} -eq 0 ]] || grep -E 'AssertionError|CypressError|Error:' "${evidence}/cypress.log" | head -5 >&2 || true
for line in ${gate[@]+"${gate[@]}"}; do info "proof gate: ${line}"; done
echo "${evidence}"
exit "${status}"
