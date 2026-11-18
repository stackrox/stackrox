#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    unset OPENSHIFT_CI
    export GITHUB_ACTION="test"
    export GITHUB_ENV="$(mktemp)"
    source "${BATS_TEST_DIRNAME}/../lib.sh"
}

function teardown() {
    rm -f "$GITHUB_ENV"
}

@test "masks a sensitive value before writing it to GITHUB_ENV" {
    run ci_export ROX_ADMIN_PASSWORD "s3cr3t"
    assert_success
    assert_output "::add-mask::s3cr3t"
    assert_equal "$(cat "$GITHUB_ENV")" "ROX_ADMIN_PASSWORD=s3cr3t"
}

@test "masks values for TOKEN and SECRET suffixes" {
    run ci_export SOME_TOKEN "tok"
    assert_output "::add-mask::tok"
    run ci_export CLIENT_SECRET "sec"
    assert_output "::add-mask::sec"
}

@test "does not mask non-sensitive values" {
    run ci_export API_ENDPOINT "localhost:8000"
    assert_success
    assert_output ""
    assert_equal "$(cat "$GITHUB_ENV")" "API_ENDPOINT=localhost:8000"
}

@test "does not mask names that only contain a sensitive word" {
    run ci_export ROX_UI_SECRETS_PAGE_MIGRATION "true"
    assert_success
    assert_output ""
}

@test "masks each line of a multi-line sensitive value" {
    run ci_export MY_SECRET $'line1\n\nline2'
    assert_success
    assert_line --index 0 "::add-mask::line1"
    assert_line --index 1 "::add-mask::line2"
    assert_equal "${#lines[@]}" 2
}

@test "skips masking an empty sensitive value" {
    run ci_export ROX_ADMIN_PASSWORD ""
    assert_success
    assert_output ""
}

@test "does not mask outside of GitHub Actions" {
    unset GITHUB_ACTION
    run ci_export ROX_ADMIN_PASSWORD "s3cr3t"
    assert_success
    refute_output --partial "add-mask"
}
