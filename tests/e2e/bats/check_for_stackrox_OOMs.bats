#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

@test "check_for_stackrox_OOMs()" {
    mkdir -p "${BATS_TEST_TMPDIR}/oom-test/stackrox"
    cp -r "${BATS_TEST_DIRNAME}/fixtures" "${BATS_TEST_TMPDIR}/oom-test/stackrox/pods"
    ARTIFACT_DIR="${BATS_TEST_TMPDIR}/artifacts"
    mkdir -p "${ARTIFACT_DIR}"

    run check_for_stackrox_OOMs "${BATS_TEST_TMPDIR}/oom-test"

    with_oomkilled_test="$(ls ${ARTIFACT_DIR}/junit-OOMCheck-central-84bf956f94-bg6hr-*.xml)"
    assert [ -f "$with_oomkilled_test" ]
    run grep -q "was OOMKilled" "$with_oomkilled_test"
    assert_success

    without_oomkilled_test="$(ls ${ARTIFACT_DIR}/junit-OOMCheck-sensor-67d98c67bf-v688m-*.xml)"
    assert [ -f "$without_oomkilled_test" ]
    run grep -q "was not OOMKilled" "$without_oomkilled_test"
    assert_success

    refute [ -f "${ARTIFACT_DIR}/junit-.xml" ]
}
