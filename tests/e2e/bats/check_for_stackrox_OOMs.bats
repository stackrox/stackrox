#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

@test "check_for_stackrox_OOMs()" {
    mkdir -p "${BATS_TEST_TMPDIR}/oom-test/stackrox"
    cp -r "${BATS_TEST_DIRNAME}/fixtures" "${BATS_TEST_TMPDIR}/oom-test/stackrox/pods"
    ARTIFACT_DIR="${BATS_TEST_TMPDIR}/artifacts"
    mkdir -p "${ARTIFACT_DIR}"

    source "${BATS_TEST_DIRNAME}/../lib.sh"

    run check_for_stackrox_OOMs "${BATS_TEST_TMPDIR}/oom-test"
    assert_success

    run cat "${ARTIFACT_DIR}/junit-misc/junit-OOM Check.xml"
    assert_success

    assert_output --partial tests=\"2\"
    assert_output --partial failures=\"1\"

    # Sensor has a non failure result
    assert_output --regexp '<testcase name="Check for sensor OOM kills" classname="OOM Check">\s+</testcase>'
    # Central has a failure result
    assert_output --regexp '<testcase name="Check for central OOM kills" classname="OOM Check">\s+<failure><..CDATA.A container of central was OOM killed..></failure>\s+</testcase>'
}
