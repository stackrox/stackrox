#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    ARTIFACT_DIR="${BATS_TEST_TMPDIR}"
    junit_dir="$(get_junit_misc_dir)"
}

@test "creates a single junit for a single test" {
    run save_junit_success "UNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="0"'
}

@test "creates multiple junit for multiple tests (different class)" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_success "OtherUNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-OtherUNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="0"'
}

@test "creates multiple junit for multiple tests (same class)" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_success "UNITTest" "Another unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="2"'
    assert_output --partial 'failures="0"'
    run grep -c 'classname="UNITTest"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 2
    run grep -c 'name="A unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
    run grep -c 'name="Another unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "is ok with repeated names" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_success "UNITTest" "Another unit test"
    run save_junit_success "UNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="3"'
    assert_output --partial 'failures="0"'
    run grep -c 'classname="UNITTest"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 3
    run grep -c 'name="A unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 2
    run grep -c 'name="Another unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "creates a failure junit" {
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="1"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "handles success and failure" {
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_success "UNITTest" "A unit test"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="2"'
    assert_output --partial 'failures="1"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "handles success and failure II" {
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_success "UNITTest" "A unit test"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="4"'
    assert_output --partial 'failures="2"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 2
}

@test "creates a skipped junit" {
    run save_junit_skipped "UNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'skipped="1"'
    assert_output --partial 'failures="0"'
}

@test "handles success, skipped and failure" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_skipped "UNITTest" "A unit test"
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_skipped "UNITTest" "A unit test"
    run save_junit_success "UNITTest" "A unit test"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="5"'
    assert_output --partial 'failures="1"'
    assert_output --partial 'skipped="2"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "handles multiline failure details" {
    read -r -d '' details << _EO_DETAILS_ || true
more failure details
other stuff
more failure details
more failure details
_EO_DETAILS_
    run save_junit_failure "UNITTest" "A unit test" "${details}"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="1"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 3
}
