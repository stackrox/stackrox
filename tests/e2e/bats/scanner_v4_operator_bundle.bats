#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

setup() {
    unset BUILD_TAG GITHUB_REF SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    TEST_ROOT="${BATS_TEST_TMPDIR}/repo"
    mkdir -p "${TEST_ROOT}/deploy/common"
    cat > "${TEST_ROOT}/deploy/common/ci-values.yaml" <<'EOF'
scannerV4:
  matcher:
customize:
  scanner-v4-matcher:
    envVars:
      SCANNER_V4_MATCHER_VULNERABILITIES_URL: https://example.invalid/ci-minimal.zip
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

@test "regular runs use the default Scanner V4 bundle allowlist" {
    run printf '%s' "${SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST}"
    assert_output "alpine,debian,epss,manual,nvd,osv,rhel-vex,stackrox-rhel-csaf,ubuntu"
}

@test "nightly runs do not set the default Scanner V4 bundle allowlist" {
    run env BUILD_TAG=4.11.x-nightly-20261001 bash -c 'unset SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST; source "$1"; [[ ! -v SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST ]]' _ "${BATS_TEST_DIRNAME}/../lib.sh"
    assert_success
}

@test "sourcing lib preserves a caller-supplied allowlist" {
    run env SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST=manual,nvd bash -c 'source "$1"; printf "%s" "$SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"' _ "${BATS_TEST_DIRNAME}/../lib.sh"
    assert_success
    assert_output "manual,nvd"
}

@test "nightly runs preserve an explicitly supplied Scanner V4 bundle allowlist" {
    export BUILD_TAG=4.11.x-nightly-20261001 SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST=manual,nvd
    run_operator_deploy
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULN_BUNDLE_ALLOWLIST")][0].value' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_success
    assert_line "manual,nvd"
}

@test "an empty caller allowlist remains empty to load all sources" {
    run env SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST= bash -c 'source "$1"; printf "%s" "$SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST"' _ "${BATS_TEST_DIRNAME}/../lib.sh"
    assert_success
    assert_output ""
}

run_operator_deploy() {
    DEPLOY_STACKROX_VIA_OPERATOR=true deploy_central_via_operator test-namespace
}

@test "nightly Operator deployments retain the production bundle default" {
    export CI=true BUILD_TAG=4.11.x-nightly-20261001
    run run_operator_deploy
    assert_success
    run yq eval-all '[select(.kind == "Central") | .spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_output "0"
}

@test "nightly Roxie deployments retain the production bundle default" {
    export CI=true GITHUB_REF=refs/tags/4.11.x-nightly-20261001
    printf 'central: {spec: {}}\n' > "${BATS_TEST_TMPDIR}/roxie.yaml"
    run _configure_roxie_ci_vuln_bundle "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_success
    run yq eval '[.central.spec.customize.envVars[]? | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/roxie.yaml"
    assert_output "0"
}

@test "Operator CI Central CR renders the minimal bundle URL" {
    export CI=true
    run run_operator_deploy
    assert_success
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")][0].value' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_line "https://example.invalid/ci-minimal.zip"
}

@test "Operator repeated invocation renders exactly one bundle URL" {
    export CI=true
    run run_operator_deploy
    assert_success
    run run_operator_deploy
    assert_success
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_line "1"
}

@test "shared CI values expose the pinned matcher URL at the customization path" {
    TEST_ROOT="$(cd "${BATS_TEST_DIRNAME}/../../.." && pwd)"
    run _scanner_v4_ci_vuln_bundle_url
    assert_success
    assert_output "https://raw.githubusercontent.com/stackrox/stackrox/4168c59464973e0c91889e3e81461d5e407aedbe/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"
}

@test "disabled Scanner V4 does not render the bundle URL" {
    export CI=true
    export ROX_SCANNER_V4=false
    : > "${TEST_ROOT}/deploy/common/ci-values.yaml"
    run run_operator_deploy
    assert_success
    run yq eval 'select(.kind == "Central") | [.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITIES_URL")] | length' "${BATS_TEST_TMPDIR}/central.yaml"
    assert_line "0"
}

@test "non-CI Operator deployment does not render the bundle URL" {
    unset CI
    : > "${TEST_ROOT}/deploy/common/ci-values.yaml"
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
customize:
  scanner-v4-matcher:
    envVars:
      SCANNER_V4_MATCHER_VULNERABILITIES_URL: null
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
    TEST_ROOT="${BATS_TEST_TMPDIR}/missing-repo"
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
    TEST_ROOT="${BATS_TEST_TMPDIR}/missing-repo"
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
customize:
  scanner-v4-matcher:
    envVars:
      SCANNER_V4_MATCHER_VULNERABILITIES_URL: null
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

@test "installation bundle values retain the pin for regular and local runs" {
    run _scanner_v4_install_bundle_values
    assert_success
    assert_output --partial "https://example.invalid/ci-minimal.zip"
}

@test "installation bundle values omit the CI override for nightlies" {
    export BUILD_TAG=4.11.x-nightly-20261001
    run _scanner_v4_install_bundle_values
    assert_success
    refute_output --partial "SCANNER_V4_MATCHER_VULNERABILITIES_URL:"
}

@test "standard Helm values omit only the CI pin at night, preserving explicit overrides" {
    source "${BATS_TEST_DIRNAME}/../../../deploy/common/k8sbased.sh"
    export BUILD_TAG=4.11.x-nightly-20261001
    _scanner_v4_ci_helm_values "${TEST_ROOT}/deploy/common/ci-values.yaml" > "${BATS_TEST_TMPDIR}/nightly.yaml"
    run yq eval '.customize."scanner-v4-matcher".envVars | has("SCANNER_V4_MATCHER_VULNERABILITIES_URL")' "${BATS_TEST_TMPDIR}/nightly.yaml"
    assert_success
    assert_output "false"
    # Helm values files merge left-to-right; no --set=null may mask this override.
    run yq eval-all '. as $item ireduce ({}; . * $item) | .customize."scanner-v4-matcher".envVars.SCANNER_V4_MATCHER_VULNERABILITIES_URL' "${BATS_TEST_TMPDIR}/nightly.yaml" "${TEST_ROOT}/deploy/common/ci-values.yaml"
    assert_success
    assert_output "https://example.invalid/ci-minimal.zip"
    unset BUILD_TAG
    _scanner_v4_ci_helm_values "${TEST_ROOT}/deploy/common/ci-values.yaml" > "${BATS_TEST_TMPDIR}/regular.yaml"
    run cmp "${BATS_TEST_TMPDIR}/regular.yaml" "${TEST_ROOT}/deploy/common/ci-values.yaml"
    assert_success
}

@test "standalone Scanner renders the pin only for regular runs and respects explicit values" {
    local chart="${BATS_TEST_DIRNAME}/../../../scanner/e2etests/helmchart"
    helm template scanner "$chart" > "${BATS_TEST_TMPDIR}/regular.yaml"
    run grep 'SCANNER_V4_MATCHER_VULNERABILITIES_URL' "${BATS_TEST_TMPDIR}/regular.yaml"
    assert_success
    helm template scanner "$chart" -f "$chart/values-full.yaml" > "${BATS_TEST_TMPDIR}/nightly.yaml"
    run grep 'SCANNER_V4_MATCHER_VULNERABILITIES_URL' "${BATS_TEST_TMPDIR}/nightly.yaml"
    assert_failure
    helm template scanner "$chart" -f "$chart/values-full.yaml" --set app.scanner.vulnerabilitiesUrl=https://example.invalid/explicit.zip > "${BATS_TEST_TMPDIR}/explicit.yaml"
    run grep 'https://example.invalid/explicit.zip' "${BATS_TEST_TMPDIR}/explicit.yaml"
    assert_success
}

@test "standalone nightly entrypoint selects production values for build tags and GitHub refs" {
    local root="${BATS_TEST_DIRNAME}/../../.."
    run env BUILD_TAG=4.11.x-nightly-20261001 make -n -C "$root/scanner" e2e-deploy
    assert_success
    assert_output --partial '-f e2etests/helmchart/values-full.yaml'
    run env GITHUB_REF=refs/tags/4.11.x-nightly-20261001 make -n -C "$root/scanner" e2e-deploy
    assert_success
    assert_output --partial '-f e2etests/helmchart/values-full.yaml'
    run make -n -C "$root/scanner" e2e-deploy
    assert_success
    refute_output --partial '-f e2etests/helmchart/values-full.yaml'
}

@test "installation bundle values fail rather than falling back when the CI pin is missing" {
    printf '{}\n' > "${TEST_ROOT}/deploy/common/ci-values.yaml"
    run _scanner_v4_install_bundle_values
    assert_failure
    assert_output --partial "CI Scanner V4 vulnerability bundle URL is empty"
}
