#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    bats_require_minimum_version 1.5.0
    repo_root="$(cd "${BATS_TEST_DIRNAME}/../../.." && pwd)"
    fixture="${BATS_TEST_TMPDIR}/repo"
    scripts="${fixture}/.github/workflows/scripts"
    mkdir -p "$scripts" "${fixture}/scanner/updater/version" "${fixture}/bin"
    cp "${repo_root}/.github/workflows/scripts/scanner-get-released-tags.sh" "$scripts/"
    unset SCANNER_BUNDLE_REFERENCE SCANNER_BUNDLE_STREAM SCANNER_RELEASE_ALLOW_RC SCANNER_BUNDLE_MASTER_REFERENCE
    export CURL_CALLS="${BATS_TEST_TMPDIR}/curl-calls"
    : > "$CURL_CALLS"
    cat > "${fixture}/scanner/updater/version/VULNERABILITY_BUNDLE_VERSION" <<'MAP'
dev heads/master
v5 heads/pinned-v5
v6 heads/master
MAP
    cat > "${scripts}/scanner-output-release-versions.sh" <<'SH'
#!/usr/bin/env bash
echo '{"versions":[]}'
SH
    cat > "${fixture}/bin/curl" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "${@: -1}" >> "$CURL_CALLS"
case "${@: -1}" in
    */pr-head/scanner/VULNERABILITY_VERSION) echo v6 ;;
    */master/scanner/VULNERABILITY_VERSION) echo "${MASTER_VERSION:-v5}" ;;
    */pinned-v5/scanner/VULNERABILITY_VERSION) echo v5 ;;
    *) exit 1 ;;
esac
SH
    chmod +x "${fixture}/bin/curl" "${scripts}/scanner-output-release-versions.sh"
    export PATH="${fixture}/bin:$PATH"
    cd "$fixture" || return 1
}

@test "PR master override validates the PR version and preserves pinned streams" {
    run --separate-stderr env SCANNER_BUNDLE_MASTER_REFERENCE=pr-head \
        "${scripts}/scanner-get-released-tags.sh"
    assert_success
    run jq -e '. == [{"version":"dev","ref":"pr-head"}, {"version":"v5","ref":"pinned-v5"}, {"version":"v6","ref":"pr-head"}]' <<< "$output"
    assert_success
    run grep '/master/scanner/' "$CURL_CALLS"
    assert_failure
}

@test "scheduled resolution rejects V6 when remote master still declares V5" {
    run --separate-stderr "${scripts}/scanner-get-released-tags.sh"
    assert_failure
    [[ "$stderr" == *"the specified version 'v6' does not match the tag's: v5"* ]]
}

@test "scheduled resolution succeeds after remote master declares V6" {
    run --separate-stderr env MASTER_VERSION=v6 "${scripts}/scanner-get-released-tags.sh"
    assert_success
    run jq -e '.[2] == {"version":"v6","ref":"master"}' <<< "$output"
    assert_success
}

@test "PR master override does not bypass version equality checks" {
    run --separate-stderr env SCANNER_BUNDLE_MASTER_REFERENCE=pinned-v5 \
        "${scripts}/scanner-get-released-tags.sh"
    assert_failure
    [[ "$stderr" == *"the specified version 'v6' does not match the tag's: v5"* ]]
}

@test "PR matrix validation supplies its head revision only to the validation step" {
    run yq eval -o=json '
        .jobs.build-matrix.steps[] |
        select(.name == "Validate matrix builder (PR)")
    ' "${repo_root}/.github/workflows/scanner-updater-schedule.yaml"
    assert_success
    run jq -e '
        .if == "github.event_name == '\''pull_request'\''" and
        .env.SCANNER_BUNDLE_MASTER_REFERENCE == "${{ github.event.pull_request.head.sha }}"
    ' <<< "$output"
    assert_success
    assert_output "true"
}
