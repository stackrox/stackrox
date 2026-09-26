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

setup_roxie_deploy_stubs() {
    export STATE_DEPLOYED="${BATS_TEST_TMPDIR}/deployed"
    gen_admin_password() { printf 'test-password\n'; }
    prepare_for_konflux() { :; }
    workaround_label_length_limitation() { :; }
    extend_roxie_envrc() { :; }
    record_build_info() { :; }
    ci_export() { :; }
    roxie() {
        local arg config_file envrc
        while (($#)); do
            arg="$1"
            case "$arg" in
                --config) config_file="$2"; shift 2 ;;
                --envrc) envrc="$2"; shift 2 ;;
                *) shift ;;
            esac
        done
        cp "$config_file" "${BATS_TEST_TMPDIR}/roxie-config.yaml"
        printf 'export ROX_DUMMY=true\n' > "$envrc"
    }
}

@test "Roxie CI entrypoint injects the pinned bundle URL" {
    setup_roxie_deploy_stubs
    export CI=true
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Enabled
EOF
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_success
    run yq eval '.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL") | .value' "${BATS_TEST_TMPDIR}/roxie-config.yaml"
    assert_output "https://example.invalid/ci-minimal.zip"
    _configure_roxie_ci_vuln_bundle "${BATS_TEST_TMPDIR}/roxie.yaml"
    _configure_roxie_ci_vuln_bundle "${BATS_TEST_TMPDIR}/roxie.yaml"
    run yq eval '[.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_output "1"
}

@test "Roxie preserves explicit valueFrom bundle URL and is idempotent" {
    export CI=true
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Enabled
    customize:
      envVars:
        - name: SCANNER_V4_MATCHER_VULNERABILITIES_URL
          valueFrom:
            secretKeyRef:
              name: bundle
              key: url
EOF
    _configure_roxie_ci_vuln_bundle "${BATS_TEST_TMPDIR}/roxie.yaml"
    _configure_roxie_ci_vuln_bundle "${BATS_TEST_TMPDIR}/roxie.yaml"
    run yq eval '[.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_output "1"
    run yq eval '.central.spec.customize.envVars[0].valueFrom.secretKeyRef.name' "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_output "bundle"
}

@test "Roxie preserves an explicit bundle URL" {
    setup_roxie_deploy_stubs
    export CI=true
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Enabled
    customize:
      envVars:
        - name: SCANNER_V4_MATCHER_VULNERABILITIES_URL
          value: https://example.invalid/custom.zip
EOF
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_success
    run yq eval '.central.spec.customize.envVars[0].value' "${BATS_TEST_TMPDIR}/roxie-config.yaml"
    assert_output "https://example.invalid/custom.zip"
}

@test "Roxie non-CI deployment does not add the bundle URL" {
    setup_roxie_deploy_stubs
    unset CI
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Enabled
EOF
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_success
    run yq eval '[.central.spec.customize.envVars[]? | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/roxie-config.yaml"
    assert_output "0"
}

@test "Roxie skips the bundle when effective Scanner V4 is disabled" {
    setup_roxie_deploy_stubs
    export CI=true ROX_SCANNER_V4=true
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Disabled
EOF
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_success
    run yq eval '[.central.spec.customize.envVars[]? | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/roxie-config.yaml"
    assert_output "0"
}

@test "Roxie fails before invoking roxie when the bundle pin is invalid" {
    setup_roxie_deploy_stubs
    export CI=true
    : > "${TEST_ROOT}/deploy/common/ci-values.yaml"
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Enabled
EOF
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_failure
    [ ! -e "${BATS_TEST_TMPDIR}/roxie-config.yaml" ]
}

@test "Roxie fails before invoking roxie when the bundle pin is null" {
    setup_roxie_deploy_stubs
    export CI=true
    cat > "${TEST_ROOT}/deploy/common/ci-values.yaml" <<'EOF'
scannerV4:
  matcher:
    vulnerabilitiesUrl: null
EOF
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Enabled
EOF
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_failure
    [ ! -e "${BATS_TEST_TMPDIR}/roxie-config.yaml" ]
}

@test "Roxie fails before invoking roxie when the bundle values file is unreadable" {
    setup_roxie_deploy_stubs
    export CI=true
    TEST_ROOT="${BATS_TEST_TMPDIR}/missing-repo"
    cat > "${BATS_TEST_TMPDIR}/roxie.yaml" <<'EOF'
central:
  spec:
    scannerV4:
      scannerComponent: Enabled
EOF
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_failure
    [ ! -e "${BATS_TEST_TMPDIR}/roxie-config.yaml" ]
}

@test "Roxie fails before invoking roxie for malformed config" {
    setup_roxie_deploy_stubs
    export CI=true
    printf 'central: [' > "${BATS_TEST_TMPDIR}/roxie.yaml"
    run deploy_stackrox_with_roxie "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_failure
    [ ! -e "${BATS_TEST_TMPDIR}/roxie-config.yaml" ]
}

@test "Roxie compatibility entrypoint injects the pinned bundle URL" {
    setup_roxie_deploy_stubs
    export CI=true MAIN_IMAGE_TAG=test-tag LOAD_BALANCER=route
    check_for_roxie() { :; }
    retrying_kubectl() { return 0; }
    run deploy_stackrox_with_roxie_compat
    assert_success
    run yq eval '.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL") | .value' "${BATS_TEST_TMPDIR}/roxie-config.yaml"
    assert_output "https://example.invalid/ci-minimal.zip"
}
