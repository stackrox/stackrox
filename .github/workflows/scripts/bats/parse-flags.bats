#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../../../scripts/test_helpers.bats"

setup() {
    source "${BATS_TEST_DIRNAME}/../lib/parse-flags.sh"
}

@test "parses a single flag with a value" {
    declare -A ARGS=()
    parse_flags ARGS --foo bar
    [[ "${ARGS[foo]}" == "bar" ]]
}

@test "parses multiple flags with values" {
    declare -A ARGS=()
    parse_flags ARGS --foo bar --baz qux
    [[ "${ARGS[foo]}" == "bar" ]]
    [[ "${ARGS[baz]}" == "qux" ]]
}

@test "a flag given without a following value is stored as an empty string" {
    declare -A ARGS=()
    parse_flags ARGS --foo
    [[ "${ARGS[foo]}" == "" ]]
}

@test "a flag immediately followed by another flag does not swallow it as a value" {
    declare -A ARGS=()
    parse_flags ARGS --foo --bar baz
    [[ "${ARGS[foo]}" == "" ]]
    [[ "${ARGS[bar]}" == "baz" ]]
}

@test "rejects an argument that is not a flag" {
    declare -A ARGS=()
    run parse_flags ARGS not-a-flag
    assert_failure
}
