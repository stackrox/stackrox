#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    bats_require_minimum_version 1.5.0
    repo_root="$(cd "${BATS_TEST_DIRNAME}/../../.." && pwd)"
    fixture="${BATS_TEST_TMPDIR}/repo"
    scripts="${fixture}/.github/workflows/scripts"
    mkdir -p "$scripts"
    cp "${repo_root}/.github/workflows/scripts/scanner-updater-matrix.sh" "$scripts/"
    cp "${repo_root}/.github/workflows/scripts/scanner-updater-config.sh" "$scripts/"
    export SOURCE_CONFIG="${fixture}/source-config.yaml"
    export FRESHNESS_CALLS="${BATS_TEST_TMPDIR}/freshness-calls"
    export SOURCE_FRESH=false
    export FRESHNESS_ERROR=false
    export SCHEDULE_TIME="2026-10-14T08:27:00Z"
    : > "$FRESHNESS_CALLS"

    cat > "$SOURCE_CONFIG" <<'YAML'
storage:
  bucket: test-bucket
  prefix: test-prefix
bundles:
  dev: [oracle, debian]
  v4: [oracle, debian]
updaters:
  default:
    interval: 4h
  oracle:
    interval: 8h
    attempt_cadence: 4h
YAML

    cat > "${scripts}/scanner-get-released-tags.sh" <<'SH'
#!/usr/bin/env bash
echo '[{"version":"dev","ref":"master"},{"version":"v4","ref":"release"}]'
SH
    cat > "${scripts}/scanner-updater-check-freshness.sh" <<'SH'
#!/usr/bin/env bash
printf '%s %s\n' "$GCS_OBJECT" "$*" >> "$FRESHNESS_CALLS"
if [[ "$FRESHNESS_ERROR" == true ]]; then
    exit 1
fi
echo "$SOURCE_FRESH"
SH
    chmod +x "${scripts}/scanner-get-released-tags.sh" \
        "${scripts}/scanner-updater-check-freshness.sh"
    cd "$fixture" || return 1
}

@test "due sources are scheduled in eligible hours across streams" {
    run "${scripts}/scanner-updater-matrix.sh"
    assert_success
    run jq -e '.has_due == "true" and .has_errors == "false" and
        ([.matrix.include[] | select(.source == "oracle")] | length == 2)' <<< "$output"
    assert_success
    run grep -c 'oracle.json.zst --max-age 28800' "$FRESHNESS_CALLS"
    assert_success
    assert_output "2"
    run grep -c 'debian.json.zst --max-age 14400' "$FRESHNESS_CALLS"
    assert_success
    assert_output "2"
}

@test "ineligible hours skip Oracle before freshness checks but retain other sources" {
    run env SCHEDULE_TIME="2026-10-14T09:27:00Z" \
        "${scripts}/scanner-updater-matrix.sh"
    assert_success
    run jq -e '.matrix.include | length == 2 and all(.[]; .source == "debian")' <<< "$output"
    assert_success
    run grep oracle "$FRESHNESS_CALLS"
    assert_failure
}

@test "fresh sources are not scheduled even in eligible hours" {
    run env SOURCE_FRESH=true "${scripts}/scanner-updater-matrix.sh"
    assert_success
    run jq -e '.has_due == "false" and .has_errors == "false" and
        (.matrix.include | length == 0)' <<< "$output"
    assert_success
}

@test "eligible windows include the whole hour and cross midnight" {
    for timestamp in 2026-10-14T23:59:59Z 2026-10-15T00:00:00Z 2026-10-15T00:59:59Z 2026-10-15T01:00:00Z; do
        expected=0
        [[ "$timestamp" == *T00:* ]] && expected=2
        run env SCHEDULE_TIME="$timestamp" "${scripts}/scanner-updater-matrix.sh"
        assert_success
        run jq -e --argjson expected "$expected" \
            '[.matrix.include[] | select(.source == "oracle")] | length == $expected' <<< "$output"
        assert_success
    done
}

@test "changing cadence changes eligible hours using UTC epoch alignment" {
    yq eval -i '.updaters.oracle.attempt_cadence = "5h"' "$SOURCE_CONFIG"
    run env SCHEDULE_TIME="1970-01-02T00:27:00Z" \
        "${scripts}/scanner-updater-matrix.sh"
    assert_success
    run jq -e '[.matrix.include[] | select(.source == "oracle")] | length == 0' <<< "$output"
    assert_success

    run env SCHEDULE_TIME="1970-01-02T01:27:00Z" \
        "${scripts}/scanner-updater-matrix.sh"
    assert_success
    run jq -e '[.matrix.include[] | select(.source == "oracle")] | length == 2' <<< "$output"
    assert_success
}

@test "freshness lookup failures remain visible scheduling errors" {
    run --separate-stderr env FRESHNESS_ERROR=true "${scripts}/scanner-updater-matrix.sh"
    assert_success
    [[ "$stderr" == *"::error::gs://test-bucket/test-prefix/dev/sources/oracle.json.zst: freshness check failed"* ]]
    run jq -e '.has_errors == "true" and (.errors | length == 4) and
        (.matrix.include | length == 0)' <<< "$output"
    assert_success
}

@test "invalid scheduling time fails rather than using the current time" {
    run env SCHEDULE_TIME="invalid" "${scripts}/scanner-updater-matrix.sh"
    assert_failure
}
