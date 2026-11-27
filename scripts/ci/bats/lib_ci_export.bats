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

@test "masks values for sensitive name patterns" {
    local name
    for name in \
        SOME_TOKEN \
        CLIENT_SECRET \
        GCP_RELEASE_AUTOMATION_SA \
        GCP_SERVICE_ACCOUNT_STACKROX_CI \
        GCLOUD_SERVICE_ACCOUNT_CI_ROX \
        GOOGLE_APPLICATION_CREDENTIALS \
        GCP_CREDENTIALS_JSON \
        ROX_LICENSE_KEY \
        GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY_V2 \
        SLACK_MAIN_WEBHOOK \
        GCP_PROJECT_NUMBER_WORKLOAD_IDENTITY; do
        run ci_export "$name" "my-secret-value"
        assert_success
        assert_output "::add-mask::my-secret-value"
    done
}

@test "masks names where a sensitive word is not at the end" {
    local name
    for name in ROX_UI_SECRETS_PAGE_MIGRATION GENERIC_WEBHOOK_SERVER_CA_CONTENTS ROX_KEYCLOAK_URL; do
        run ci_export "$name" "my-secret-value"
        assert_success
        assert_output "::add-mask::my-secret-value"
    done
}

@test "does not mask placeholder values of sensitive names" {
    local value
    for value in DISABLED true False enabled none null On off AUTO '[]' '{}'; do
        run ci_export ROX_TELEMETRY_STORAGE_KEY_V1 "$value"
        assert_success
        assert_output ""
    done
}

@test "does not mask non-sensitive values" {
    run ci_export API_ENDPOINT "localhost:8000"
    assert_success
    assert_output ""
    assert_equal "$(cat "$GITHUB_ENV")" "API_ENDPOINT=localhost:8000"
}

@test "masks a multi-line sensitive value as a single escaped mask" {
    run ci_export MY_SECRET $'line1\n\nline2'
    assert_success
    assert_output "::add-mask::line1%0A%0Aline2"
}

@test "escapes workflow-command data in the mask but not in GITHUB_ENV" {
    run ci_export ROX_ADMIN_PASSWORD 'pfx%25sfx!'
    assert_success
    assert_output '::add-mask::pfx%2525sfx!'
    assert_equal "$(cat "$GITHUB_ENV")" 'ROX_ADMIN_PASSWORD=pfx%25sfx!'
}

@test "escapes carriage returns in the mask" {
    run ci_export ROX_ADMIN_PASSWORD $'a\rb'
    assert_success
    assert_output '::add-mask::a%0Db'
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
