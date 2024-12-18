#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    ARTIFACT_DIR="${BATS_TEST_TMPDIR}"
}

@test "service_get_endpoint gets IP from endpoint when IP is reported" {
    run service_get_endpoint < "${BATS_TEST_DIRNAME}/fixtures/service-ip.json"
    assert_success
    assert_output "35.193.73.252"
}

@test "service_get_endpoint gets hostname from endpoint when hostname is reported" {
    run service_get_endpoint < "${BATS_TEST_DIRNAME}/fixtures/service-hostname.json"
    assert_success
    assert_output "a210901dccf824daf8118d5aa9993115-2055499247.us-east-2.elb.amazonaws.com"
}

@test "service_get_endpoint gets hostname from endpoint when both hostname and ip is reported" {
    run service_get_endpoint < "${BATS_TEST_DIRNAME}/fixtures/service-both.json"
    assert_success
    assert_output "a210901dccf824daf8118d5aa9993115-2055499247.us-east-2.elb.amazonaws.com"
}

@test "service_get_endpoint gets first endpoint if multiple are reported" {
    run service_get_endpoint < "${BATS_TEST_DIRNAME}/fixtures/service-two-ingresses.json"
    assert_success
    assert_output "35.193.73.252"
}

@test "service_get_endpoint fails with informative error when neither hostname nor IP is reported" {
    run service_get_endpoint < "${BATS_TEST_DIRNAME}/fixtures/service-unready.json"
    assert_failure
    # This actually goes to stderr, but bats-helpers do not seem
    # to distinguish between stderr and stdout.
    assert_output --partial "List of ingress points of LB stackrox-images-metrics is empty."
}
