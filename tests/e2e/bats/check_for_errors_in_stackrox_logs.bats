#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../../scripts/test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    function mock_check_script() {
        # shellcheck disable=SC2317
        >&2 echo "check called with: $*<<"
        # shellcheck disable=SC2317
        true
    }
    function save_junit_success() {
        # shellcheck disable=SC2317
        >&2 echo "save_junit_success called with: $*<<"
        # shellcheck disable=SC2317
        true
    }
}

@test "expects args" {
    run check_for_errors_in_stackrox_logs
    assert_failure
    assert_output --partial 'missing args'
}

@test "expects logs" {
    run check_for_errors_in_stackrox_logs "${BATS_TEST_TMPDIR}"
    assert_failure
    assert_output --partial 'logs were not collected'
}

@test "expects a count of items" {
    mkdir -p "${BATS_TEST_TMPDIR}/stackrox/pods"
    run check_for_errors_in_stackrox_logs "${BATS_TEST_TMPDIR}"
    assert_failure
    assert_output --partial 'ITEM_COUNT.txt is missing'
}

@test "count of items != file count (none)" {
    mkdir -p "${BATS_TEST_TMPDIR}/stackrox/pods"
    echo 1 > "${BATS_TEST_TMPDIR}/stackrox/pods/ITEM_COUNT.txt"
    run check_for_errors_in_stackrox_logs "${BATS_TEST_TMPDIR}"
    assert_failure
    assert_output --partial 'The recorded number of items (1) differs from the objects found (0)'
}

@test "count of items != file count (2)" {
    mkdir -p "${BATS_TEST_TMPDIR}/stackrox/pods"
    echo 1 > "${BATS_TEST_TMPDIR}/stackrox/pods/ITEM_COUNT.txt"
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/a_object.json"
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/b_object.json"
    run check_for_errors_in_stackrox_logs "${BATS_TEST_TMPDIR}"
    assert_failure
    assert_output --partial 'The recorded number of items (1) differs from the objects found (2)'
}

@test "check.sh will be called multiple times with combined logs for the same app" {
    mkdir -p "${BATS_TEST_TMPDIR}/stackrox/pods"
    echo 5 > "${BATS_TEST_TMPDIR}/stackrox/pods/ITEM_COUNT.txt"
    make_pod_object "an_app-a" "an_app"
    make_pod_object "an_app-b" "an_app"
    make_pod_object "other_app-c" "other_app"
    make_pod_object "no_logs_app-d" "no_logs_app"
    make_pod_object "no_logs_app-e" "no_logs_app"
    # multiple containers per pod
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/an_app-a-1.log"
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/an_app-a-2.log"
    # mulptiple pods per app
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/an_app-b.log"
    # a log from a previously successful container instance
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/other_app-c-prev-success.log"
    # describe yaml saved to a .log file is ignored
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/other_app-c_describe.log"
    # logs from previous container failures are ignored (handled elsewhere)
    touch "${BATS_TEST_TMPDIR}/stackrox/pods/other_app-c_previous.log"
    LOGCHECK_SCRIPT="mock_check_script"

    run check_for_errors_in_stackrox_logs "${BATS_TEST_TMPDIR}"
    assert_success

    assert_output --partial "\
check called with: ${BATS_TEST_TMPDIR}/stackrox/pods/an_app-a-1.log \
${BATS_TEST_TMPDIR}/stackrox/pods/an_app-a-2.log \
${BATS_TEST_TMPDIR}/stackrox/pods/an_app-b.log<<"
    assert_output --partial "save_junit_success called with: SuspiciousLog-an_app "

    assert_output --partial "\
check called with: ${BATS_TEST_TMPDIR}/stackrox/pods/other_app-c-prev-success.log<<"
    assert_output --partial "save_junit_success called with: SuspiciousLog-other_app "

    assert_output --partial "no_logs_app-d*.log': No such file or directory"
    assert_output --partial "no_logs_app-e*.log': No such file or directory"
    assert_output --partial "save_junit_success called with: SuspiciousLog-no_logs_app "
}

function make_pod_object() {
    cat <<__POD_OBJECT__ > "${BATS_TEST_TMPDIR}/stackrox/pods/$1_object.json"
{
    "metadata": {
        "name": "$1",
        "labels": {
            "app": "$2"
        }
    }
}
__POD_OBJECT__
}
