#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    repo_root="$(cd "${BATS_TEST_DIRNAME}/../../.." && pwd)"
    config_script="${repo_root}/.github/workflows/scripts/scanner-updater-config.sh"
    export SOURCE_CONFIG="${BATS_TEST_TMPDIR}/source-config.yaml"
}

@test "Oracle inherits default freshness without an attempt cadence" {
    run env SOURCE_CONFIG="${repo_root}/scanner/updater/ci/source-config.yaml" \
        "$config_script" interval-secs oracle
    assert_success
    assert_output "14400"

    run env SOURCE_CONFIG="${repo_root}/scanner/updater/ci/source-config.yaml" \
        "$config_script" attempt-cadence-secs oracle
    assert_success
    assert_output "0"
}

@test "sources without a cadence retain default freshness and unrestricted attempts" {
    run env SOURCE_CONFIG="${repo_root}/scanner/updater/ci/source-config.yaml" \
        "$config_script" interval-secs debian
    assert_success
    assert_output "14400"

    run env SOURCE_CONFIG="${repo_root}/scanner/updater/ci/source-config.yaml" \
        "$config_script" attempt-cadence-secs debian
    assert_success
    assert_output "0"
}

@test "attempt cadence inherits the default and supports per-source overrides" {
    cat > "$SOURCE_CONFIG" <<'YAML'
updaters:
  default:
    attempt_cadence: 6h
  oracle:
    attempt_cadence: 240m
YAML
    run "$config_script" attempt-cadence-secs debian
    assert_success
    assert_output "21600"

    run "$config_script" attempt-cadence-secs oracle
    assert_success
    assert_output "14400"
}

@test "attempt cadence rejects zero and fractional-hour durations" {
    for cadence in 0h 30m 1h30m; do
        printf 'updaters:\n  oracle:\n    attempt_cadence: %s\n' "$cadence" > "$SOURCE_CONFIG"
        run "$config_script" attempt-cadence-secs oracle
        assert_failure
        assert_output --partial "attempt_cadence must be a positive whole-hour duration"
    done
}

@test "attempt cadence rejects malformed durations" {
    printf 'updaters:\n  oracle:\n    attempt_cadence: invalid\n' > "$SOURCE_CONFIG"
    run "$config_script" attempt-cadence-secs oracle
    assert_failure
    assert_output --partial "cannot parse duration"
}
