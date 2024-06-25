#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"
bats_require_minimum_version 1.5.0

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
    assert_output --partial '<skipped/>'
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

@test "escapes XML in name - double quote" {
    run save_junit_failure 'UNITTest' '"A unit test"' "nada"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'name="&quot;A unit test&quot;"'
}

@test "escapes XML in name - single quote" {
    run save_junit_failure 'UNITTest' "'A unit test'" "nada"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'name="&#39;A unit test&#39;"'
}

@test "escapes XML in name - <>&" {
    run save_junit_failure 'UNITTest' "A <unit> &test" "nada"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'name="A &lt;unit&gt; &amp;test"'
}

@test "JUNIT wrapping functionality" {
    wrapped_functionality() {
        echo "This bit of fn() will have a JUNIT record \o/"
    }
    run junit_wrap "Suite" "Test" "failure message" "wrapped_functionality"
    assert_success
    run cat "${junit_dir}/junit-Suite.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="0"'
}

@test "JUNIT wrapping functionality - propagates failure" {
    wrapped_functionality() {
        not_a_valid_command
    }
    run -127 junit_wrap "Suite" "Test" "failure message" "wrapped_functionality"
    assert_failure
    run cat "${junit_dir}/junit-Suite.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="1"'
}

@test "JUNIT wrapping functionality - propagates exports" {
    BEFORE="before"
    wrapped_functionality() {
        export BEFORE="after"
    }
    junit_wrap "Suite" "Test" "failure message" "wrapped_functionality"
    assert_equal "${BEFORE}" "after"
}

@test "JUNIT wrapping functionality - includes output on failure" {
    wrapped_functionality() {
        echo "This bit of fn() will have a JUNIT record \o/"
        not_a_valid_command
    }
    run -127 junit_wrap "Suite" "Test" "failure message" "wrapped_functionality"
    run cat "${junit_dir}/junit-Suite.xml"
    assert_output --partial "This bit of fn() will have a JUNIT record \o/"
}

@test "JUNIT wrapping functionality - includes stderr on failure" {
    wrapped_functionality() {
        >&2 echo "This bit of fn() will have a JUNIT record \o/"
        ls -l /noexisto
    }
    run junit_wrap "Suite" "Test" "failure message" "wrapped_functionality"
    run cat "${junit_dir}/junit-Suite.xml"
    assert_output --partial "This bit of fn() will have a JUNIT record \o/"
    assert_output --partial "ls: cannot access '/noexisto': No such file or directory"
}
