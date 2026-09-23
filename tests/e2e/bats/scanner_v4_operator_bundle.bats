#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    TEST_ROOT="${BATS_TEST_TMPDIR}/repo"
    mkdir -p "${TEST_ROOT}/deploy/common"
    cat > "${TEST_ROOT}/deploy/common/ci-values.yaml" <<'EOF'
scannerV4:
  matcher:
    vulnerabilitiesUrl: https://example.invalid/ci-minimal.zip
EOF
    printf 'key' > "${BATS_TEST_TMPDIR}/tls.key"
    printf 'cert' > "${BATS_TEST_TMPDIR}/tls.crt"
    : > "${BATS_TEST_TMPDIR}/trusted-ca"
    export ROX_DEFAULT_TLS_KEY_FILE="${BATS_TEST_TMPDIR}/tls.key"
    export ROX_DEFAULT_TLS_CERT_FILE="${BATS_TEST_TMPDIR}/tls.crt"
    export TRUSTED_CA_FILE="${BATS_TEST_TMPDIR}/trusted-ca"
    export ROX_SCANNER_V4=true
    export ROX_BASELINE_GENERATION_DURATION=10s
    export ROX_NETWORK_BASELINE_OBSERVATION_PERIOD=10s
    export ROX_SENSITIVE_FILE_ACTIVITY=false ROX_CVE_FIX_TIMESTAMP=true
    export ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL=true ROX_UI_SECRETS_PAGE_MIGRATION=false
    export ROX_AI_INTEGRATIONS=false ROX_LIGHTSPEED_RISK_SUMMARY=false
    export ROX_BASE_IMAGE_DETECTION=false ROX_VULNERABILITY_VIEW_BASED_REPORTS=true
    export ROX_NETWORK_GRAPH_EXTERNAL_IPS=false ROX_VIRTUAL_MACHINES=false
    export LOAD_BALANCER=none
    export USE_MIDSTREAM_IMAGES=false
    make() { :; }
    retrying_kubectl() {
        if [[ " $* " == *" apply "* ]]; then
            cat > "${BATS_TEST_TMPDIR}/central.yaml"
        fi
    }
    wait_for_object_to_appear() { :; }
}

run_operator_deploy() {
    DEPLOY_STACKROX_VIA_OPERATOR=true deploy_central_via_operator test-namespace
}

@test "Operator CI Central CR renders the minimal bundle URL" {
    export CI=true
    run run_operator_deploy
    assert_success
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")][0].value' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_line "https://example.invalid/ci-minimal.zip"
}

@test "disabled Scanner V4 does not render the bundle URL" {
    export CI=true
    export ROX_SCANNER_V4=false
    run run_operator_deploy
    assert_success
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_line "0"
}

@test "non-CI Operator deployment does not render the bundle URL" {
    unset CI
    run run_operator_deploy
    assert_success
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_line "0"
}

@test "midstream Operator Central CR renders the minimal bundle URL" {
    export CI=true USE_MIDSTREAM_IMAGES=true
    run run_operator_deploy
    assert_success
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")][0].value' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_line "https://example.invalid/ci-minimal.zip"
}

@test "CI deployment fails when the bundle URL is missing" {
    export CI=true
    : > "${TEST_ROOT}/deploy/common/ci-values.yaml"
    run run_operator_deploy
    assert_failure
    assert_output --partial "CI Scanner V4 vulnerability bundle URL is empty"
}

@test "CI deployment fails when the bundle URL is null" {
    export CI=true
    cat > "${TEST_ROOT}/deploy/common/ci-values.yaml" <<'EOF'
scannerV4:
  matcher:
    vulnerabilitiesUrl: null
EOF
    run run_operator_deploy
    assert_failure
    assert_output --partial "CI Scanner V4 vulnerability bundle URL is empty"
}

@test "CI deployment fails when the shared values file cannot be read" {
    export CI=true
    TEST_ROOT="${BATS_TEST_TMPDIR}/missing-repo"
    run run_operator_deploy
    assert_failure
    assert_output --partial "Unable to read the CI Scanner V4 vulnerability bundle URL"
}
