#!/usr/bin/env bats

load "./test_helpers.bats"

setup() {
    program="$BATS_TEST_DIRNAME"/get-previous-y-stream.sh
}

@test "when called without arguments, returns no-zero and prints usage" {
    run "$program"
    assert_failure
    assert_output --partial "Usage:"
}

@test "when asked for help, prints usage" {
    run "$program" --help
    assert_output --partial "Usage:"
}

@test "when called with bogus input, fails with error" {
    test_invalid "13"
    test_invalid "a.b.c"
    test_invalid "3.0.62.x"
    test_invalid "3.0.62.1"
    test_invalid " v4.0.0"
    test_invalid ".."
    test_invalid "..-1.2.3"
    test_invalid "1.2.3-"
}

test_invalid() {
    run "$program" "$1"
    assert_failure
    assert_output --partial "Error:"
    assert_output --partial "$1"
}

@test "when provided more arguments than just one, fails with error" {
    test_excessive_args "1.2.3" "4.5.6"
    test_excessive_args "1.2.3" "--help"
    test_excessive_args "foo" "bar" "baz"
}

test_excessive_args() {
    run "$program" "$@"
    assert_failure
    assert_output --regexp "Error:.*too many.*arguments"
}

@test "when called with unknown new major, fails with error" {
    test_major_unknown "0.0.0" "0.0"
    test_major_unknown "2.0.0" "2.0"
    test_major_unknown "3.0.0" "3.0"
    test_major_unknown "5.0.0" "5.0"
    test_major_unknown "5.0.1" "5.0"
    test_major_unknown "v5.0.x-nightly-12345" "5.0"
    test_major_unknown "199.0.88" "199.0"
}

test_major_unknown() {
    run "$program" "$1"
    assert_failure
    assert_output --partial "Error:"
    assert_output --partial "$2"
}

@test "when called with known major, prints expected previous release" {
    test_happy "4.0.0" "3.74.0"
    test_happy "4.0.6" "3.74.0"
    test_happy "4.0.x-12-g8e6387" "3.74.0"
    test_happy "1.0.0" "0.0.0"
    test_happy "v1.0.0" "0.0.0"
}

@test "when called with ordinary minor, prints expected previous release" {
    test_happy "3.74.x-nightly-20230224" "3.73.0"
    test_happy "3.74.x-609-g3fc7d38fdf" "3.73.0"
    test_happy "3.70.0-609-g3fc7d38fdf" "3.69.0"
    test_happy "4.1.0" "4.0.0"
    test_happy "5.10.2" "5.9.0"
    test_happy "v3.62.7" "3.61.0"
    test_happy "45.67.89" "45.66.0"
}

test_happy() {
    run "$program" "$1"
    assert_success
    assert_output "$2"
}
