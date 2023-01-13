#!/usr/bin/env bats

# load "../../scripts/test_helpers.bats"
load '/opt/homebrew/lib/bats-support/load.bash'
load '/opt/homebrew/lib/bats-assert/load.bash'
load '/opt/homebrew/lib/bats-file/load.bash'

function setup() {
    TEST_TEMP_DIR="$(temp_make)"
    BATSLIB_FILE_PATH_REM=''
    BATSLIB_FILE_PATH_ADD=''
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "check_for_stackrox_OOMs()" {
    mkdir -p "${BATS_TEST_TMPDIR}/oom-test/stackrox"
    cp -r "${BATS_TEST_DIRNAME}/fixtures" "${BATS_TEST_TMPDIR}/oom-test/stackrox/pods"
    ARTIFACT_DIR="${TEST_TEMP_DIR}/artifacts"
    mkdir -p "${ARTIFACT_DIR}"
    run check_for_stackrox_OOMs "${BATS_TEST_TMPDIR}/oom-test"
    with_oomkilled_test="${ARTIFACT_DIR}/junit-OOMCheck-central-84bf956f94-bg6hr.xml"
    assert_file_exist "$with_oomkilled_test" 
    assert_file_contains "$with_oomkilled_test" "was OOMKilled"
    without_oomkilled_test="${ARTIFACT_DIR}/junit-OOMCheck-sensor-67d98c67bf-v688m.xml"
    assert_file_exist "$without_oomkilled_test"
    assert_file_contains "$without_oomkilled_test" "was not OOMKilled"
    assert_not_exist "${ARTIFACT_DIR}/junit-.xml"
    assert_success
}