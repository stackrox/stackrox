#!/usr/bin/env bats

# shellcheck disable=SC2329 # roxcurl is invoked by the sourced preflight helper.
load "../../../scripts/test_helpers.bats"

setup() {
    source "${BATS_TEST_DIRNAME}/../../upgrade/lib.sh"
    export SCANNER_V4_SCAN_READINESS_TIMEOUT=30
    export preflight_attempts="${BATS_TEST_TMPDIR}/attempts"
    echo 0 >"${preflight_attempts}"
}

info() {
    echo "$*"
}

die() {
    echo "$*" >&2
    exit 1
}

# sleep replaces the preflight's retry delay in these tests. Advancing SECONDS,
# which the preflight uses for its deadline, lets timeout tests reach that
# deadline without waiting in real time.
sleep() {
    SECONDS=$((SECONDS + $1))
}

@test "scanner preflight retries until a fixable CVE has CVSS at least seven" {
    export SCANNER_V4_SCAN_READINESS_TIMEOUT=60
    roxcurl() {
        local attempt
        attempt="$(cat "${preflight_attempts}")"
        echo "$((attempt + 1))" >"${preflight_attempts}"
        if [[ "$*" != *'"force": true'* || "$1" != /v1/images/scan ]]; then
            echo "invalid scan request" >&2
            return 1
        fi
        case "$attempt" in
            0) echo 'rpc error: code = Unavailable desc = no children to pick from'; return 1 ;;
            1) echo '{"scan":{"scanTime":"2026-10-21T00:00:00Z","components":[{"vulns":[{"fixedBy":"2.0","cvss":0}]}]}}' ;;
            2) echo '{"scan":{"scanTime":"2026-10-21T00:00:00Z","components":[{"vulns":[{"fixedBy":"2.0","cvss":6.9}]}]}}' ;;
            3) echo '{"scan":{"scanTime":"2026-10-21T00:00:00Z","components":[{"vulns":[{"cvss":9.8},{"fixedBy":"2.0","cvss":6.9}]}]}}' ;;
            *) echo '{"scan":{"scanTime":"2026-10-21T00:00:00Z","components":[{"vulns":[{"fixedBy":"2.0","cvss":7}]}]}}' ;;
        esac
    }

    run wait_for_scored_image_scan example.com/preflight:latest
    assert_success
    assert_output --partial "no children to pick from"
    assert_output --partial "no fixable CVEs with CVSS >= 7 yet"
    assert_output --partial "completed the scanner preflight"
    [[ "$(cat "${preflight_attempts}")" == 5 ]]
}

@test "scanner preflight retries invalid responses and requires a completed scan" {
    roxcurl() {
        local attempt
        attempt="$(cat "${preflight_attempts}")"
        echo "$((attempt + 1))" >"${preflight_attempts}"
        case "$attempt" in
            0) echo '<html>Bad Gateway</html>' ;;
            1) echo '{"scan":{"components":[{"vulns":[{"fixedBy":"2.0","cvss":9.8}]}]}}' ;;
            *) echo '{"scan":{"scanTime":"2026-10-21T00:00:00Z","components":[{"vulns":[{"fixedBy":"2.0","cvss":9.8}]}]}}' ;;
        esac
    }

    run wait_for_scored_image_scan example.com/preflight:latest
    assert_success
    assert_output --partial "invalid scan response"
    [[ "$(cat "${preflight_attempts}")" == 3 ]]
}

@test "scanner preflight fails within its deadline when scores remain unavailable" {
    roxcurl() {
        echo '{"scan":{"scanTime":"2026-10-21T00:00:00Z","components":[]}}'
    }

    run wait_for_scored_image_scan example.com/preflight:latest
    assert_failure
    assert_output --partial "within 30s"
}

@test "scanner preflight reports registry errors separately from policy failures" {
    roxcurl() {
        echo '{"message":"registry returned 502 Bad Gateway"}'
        return 22
    }

    run wait_for_scored_image_scan example.com/preflight:latest
    assert_failure
    assert_output --partial "registry returned 502 Bad Gateway"
}
