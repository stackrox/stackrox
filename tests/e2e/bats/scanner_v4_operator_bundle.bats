#!/usr/bin/env bats

load "../../../scripts/test_helpers.bats"

setup() {
    unset BUILD_TAG GITHUB_REF SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST SCANNER_V4_FULL_BUNDLE
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    TEST_ROOT="${BATS_TEST_TMPDIR}/repo"
    mkdir -p "${TEST_ROOT}/deploy/common"
    printf '{}\n' > "${TEST_ROOT}/deploy/common/ci-values.yaml"
    mkdir -p "$TEST_ROOT/scanner/image/scanner/bundles/ci-minimal"
    printf 'fixture' > "$TEST_ROOT/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"
    EXPECTED_VERSION="$(scanner_v4_ci_bundle_version "$TEST_ROOT")"
    is_in_PR_context() { return 1; }
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

@test "Central install selects bundles for current and older charts" {
    for chart_case in current older; do
        local chart_dir="" expected=""
        if [[ "$chart_case" == "older" ]]; then
            chart_dir="${BATS_TEST_TMPDIR}/old-central-chart"
            expected=""
        fi
        local capture="${BATS_TEST_TMPDIR}/central-${chart_case}-chart-arg"
        run exercise_central_install_bundle_call "$chart_dir" "$capture"
        assert_failure
        run cat "$capture"
        assert_output "$expected"
    done
}

@test "Sensor install selects bundles for current and older charts" {
    for chart_case in current older; do
        local chart_dir="" expected=""
        if [[ "$chart_case" == "older" ]]; then
            chart_dir="${BATS_TEST_TMPDIR}/old-sensor-chart"
            expected=""
        fi
        local capture="${BATS_TEST_TMPDIR}/sensor-${chart_case}-chart-arg"
        run exercise_sensor_install_bundle_call "$chart_dir" "$capture"
        assert_failure
        run cat "$capture"
        assert_output "$expected"
    done
}

exercise_central_install_bundle_call() {
    local chart_dir="$1" capture="$2"
    local install_script="${BATS_TEST_DIRNAME}/../run-scanner-v4-install.bats"
    eval "$(sed -n '/^deploy_central_with_helm() {/,/^}/p' "$install_script")"
    _scanner_v4_install_bundle_values() { printf '%s\n' "${1:-}" > "$capture"; return 1; }
    helm() { return 1; }
    jq() { return 1; }
    create_central_pull_secrets() { :; }
    export HEAD_HELM_CHART_CENTRAL_SERVICES_DIR="${BATS_TEST_TMPDIR}/head-central-chart"
    export DEFAULT_IMAGE_REGISTRY="example.invalid"
    deploy_central_with_helm test-central test-tag "$chart_dir"
}

exercise_sensor_install_bundle_call() {
    local chart_dir="$1" capture="$2"
    local install_script="${BATS_TEST_DIRNAME}/../run-scanner-v4-install.bats"
    eval "$(sed -n '/^deploy_sensor_with_helm() {/,/^}/p' "$install_script")"
    _scanner_v4_install_bundle_values() { printf '%s\n' "${1:-}" > "$capture"; return 1; }
    helm() { return 1; }
    jq() { return 1; }
    create_sensor_pull_secrets() { :; }
    export HEAD_HELM_CHART_SECURED_CLUSTER_SERVICES_DIR="${BATS_TEST_TMPDIR}/head-sensor-chart"
    ORCH_CMD="${BATS_TEST_TMPDIR}/orch"
    cat > "$ORCH_CMD" <<'EOF'
#!/usr/bin/env bash
printf 'init-bundle\n'
EOF
    chmod +x "$ORCH_CMD"
    deploy_sensor_with_helm test-central test-sensor test-tag "$chart_dir" test-cluster test-password test-endpoint
}

@test "regular Operator CI selects the hashed bundle through Central" {
    export CI=true
    run run_operator_deploy
    assert_success
    run yq eval '.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITY_VERSION") | .value' "$BATS_TEST_TMPDIR/central.yaml"
    assert_output "$EXPECTED_VERSION"
    run grep SCANNER_V4_MATCHER_VULNERABILITIES_URL "$BATS_TEST_TMPDIR/central.yaml"
    assert_failure
}

@test "Operator full modes and disabled Scanner retain production defaults" {
    export CI=true
    for mode in nightly environment label disabled non-ci; do
        unset BUILD_TAG SCANNER_V4_FULL_BUNDLE
        export CI=true ROX_SCANNER_V4=true
        is_in_PR_context() { return 1; }
        case "$mode" in
            nightly) export BUILD_TAG=5.1.x-nightly-20261020 ;;
            environment) export SCANNER_V4_FULL_BUNDLE=true ;;
            label) is_in_PR_context() { return 0; }; pr_has_label() { [[ "$1" == ci-scanner-full-bundle ]]; } ;;
            disabled) export ROX_SCANNER_V4=false ;;
            non-ci) unset CI ;;
        esac
        run run_operator_deploy
        assert_success
        run grep SCANNER_V4_MATCHER_VULNERABILITY_VERSION "$BATS_TEST_TMPDIR/central.yaml"
        assert_failure
    done
}

@test "midstream Operator CI selects the same bundle version" {
    export CI=true USE_MIDSTREAM_IMAGES=true
    run run_operator_deploy
    assert_success
    run grep "$EXPECTED_VERSION" "$BATS_TEST_TMPDIR/central.yaml"
    assert_success
}

@test "CI does not automatically restrict bundle sources" {
    [[ ! -v SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST ]]
    export CI=true
    run run_operator_deploy
    assert_success
    run grep SCANNER_V4_MATCHER_VULN_BUNDLE_ALLOWLIST "$BATS_TEST_TMPDIR/central.yaml"
    assert_failure
}

@test "explicit allowlists remain effective for full-bundle testing" {
    export CI=true SCANNER_V4_FULL_BUNDLE=true SCANNER_V4_CI_VULN_BUNDLE_ALLOWLIST=manual,nvd
    run run_operator_deploy
    assert_success
    run yq eval '.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULN_BUNDLE_ALLOWLIST") | .value' "$BATS_TEST_TMPDIR/central.yaml"
    assert_output manual,nvd
}

@test "missing ZIP fails Operator deployment" {
    export CI=true
    TEST_ROOT="$BATS_TEST_TMPDIR/missing"
    run run_operator_deploy
    assert_failure
    assert_output --partial 'Missing or empty Scanner CI bundle'
}

@test "Roxie CI injects the version exactly once" {
    setup_roxie_deploy_stubs
    export CI=true
    printf 'central: {spec: {scannerV4: {scannerComponent: Enabled}}}\n' > "$BATS_TEST_TMPDIR/roxie.yaml"
    run deploy_stackrox_with_roxie "$BATS_TEST_TMPDIR/roxie.yaml"
    assert_success
    _configure_roxie_ci_vuln_bundle "$BATS_TEST_TMPDIR/roxie.yaml"
    _configure_roxie_ci_vuln_bundle "$BATS_TEST_TMPDIR/roxie.yaml"
    run yq eval '[.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITY_VERSION")] | length' "$BATS_TEST_TMPDIR/roxie.yaml"
    assert_output 1
    run yq eval '.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITY_VERSION") | .value' "$BATS_TEST_TMPDIR/roxie.yaml"
    assert_output "$EXPECTED_VERSION"
}

@test "Roxie preserves explicit version and URL valueFrom overrides" {
    export CI=true
    for setting in SCANNER_V4_MATCHER_VULNERABILITY_VERSION SCANNER_V4_MATCHER_VULNERABILITIES_URL; do
        printf 'central:\n  spec:\n    customize:\n      envVars:\n        - name: %s\n          valueFrom:\n            secretKeyRef: {name: bundle, key: value}\n' "$setting" > "$BATS_TEST_TMPDIR/roxie.yaml"
        TEST_ROOT="$BATS_TEST_TMPDIR/missing"
        run _configure_roxie_ci_vuln_bundle "$BATS_TEST_TMPDIR/roxie.yaml"
        assert_success
        run yq eval '.central.spec.customize.envVars | length' "$BATS_TEST_TMPDIR/roxie.yaml"
        assert_output 1
    done
}

@test "Roxie full modes and disabled components omit the minimal override" {
    export CI=true
    for mode in nightly environment disabled non-ci; do
        unset BUILD_TAG SCANNER_V4_FULL_BUNDLE
        export CI=true
        printf 'central: {spec: {}}\n' > "$BATS_TEST_TMPDIR/roxie.yaml"
        case "$mode" in
            nightly) export BUILD_TAG=5.1.x-nightly-20261020 ;;
            environment) export SCANNER_V4_FULL_BUNDLE=true ;;
            disabled) printf 'central: {spec: {scannerV4: {scannerComponent: Disabled}}}\n' > "$BATS_TEST_TMPDIR/roxie.yaml" ;;
            non-ci) unset CI ;;
        esac
        run _configure_roxie_ci_vuln_bundle "$BATS_TEST_TMPDIR/roxie.yaml"
        assert_success
        run grep SCANNER_V4_MATCHER_VULNERABILITY_VERSION "$BATS_TEST_TMPDIR/roxie.yaml"
        assert_failure
    done
}

@test "Roxie fails before deployment on missing bundle or malformed config" {
    setup_roxie_deploy_stubs
    export CI=true
    printf 'central: {spec: {}}\n' > "$BATS_TEST_TMPDIR/roxie.yaml"
    TEST_ROOT="$BATS_TEST_TMPDIR/missing"
    run deploy_stackrox_with_roxie "$BATS_TEST_TMPDIR/roxie.yaml"
    assert_failure
    [[ ! -e "$BATS_TEST_TMPDIR/roxie-config.yaml" ]]
    printf 'central: [' > "$BATS_TEST_TMPDIR/roxie.yaml"
    run _configure_roxie_ci_vuln_bundle "$BATS_TEST_TMPDIR/roxie.yaml"
    assert_failure
}

@test "current and older installation charts select a version; full mode omits it" {
    run _scanner_v4_install_bundle_values true
    assert_success
    assert_output --partial "$EXPECTED_VERSION"
    refute_output --partial SCANNER_V4_MATCHER_VULNERABILITIES_URL
    run _scanner_v4_install_bundle_values false
    assert_success
    assert_output --partial "$EXPECTED_VERSION"
    export SCANNER_V4_FULL_BUNDLE=true
    run _scanner_v4_install_bundle_values true
    assert_success
    refute_output --partial SCANNER_V4_MATCHER_VULNERABILITY_VERSION
}

@test "Helm CI values select a version and preserve explicit overrides" {
    source "${BATS_TEST_DIRNAME}/../../../deploy/common/k8sbased.sh"
    _scanner_v4_ci_helm_values "$TEST_ROOT/deploy/common/ci-values.yaml" > "$BATS_TEST_TMPDIR/rendered.yaml"
    run yq eval '.customize."scanner-v4-matcher".envVars.SCANNER_V4_MATCHER_VULNERABILITY_VERSION' "$BATS_TEST_TMPDIR/rendered.yaml"
    assert_output "$EXPECTED_VERSION"
    for setting in SCANNER_V4_MATCHER_VULNERABILITY_VERSION SCANNER_V4_MATCHER_VULNERABILITIES_URL; do
        printf 'customize:\n  scanner-v4-matcher:\n    envVars:\n      %s: explicit\n' "$setting" > "$TEST_ROOT/deploy/common/ci-values.yaml"
        _scanner_v4_ci_helm_values "$TEST_ROOT/deploy/common/ci-values.yaml" > "$BATS_TEST_TMPDIR/rendered.yaml"
        run cmp "$TEST_ROOT/deploy/common/ci-values.yaml" "$BATS_TEST_TMPDIR/rendered.yaml"
        assert_success
    done
}

@test "Helm full modes and disabled Scanner omit the minimal override" {
    source "${BATS_TEST_DIRNAME}/../../../deploy/common/k8sbased.sh"
    for mode in nightly environment disabled; do
        unset BUILD_TAG SCANNER_V4_FULL_BUNDLE
        export ROX_SCANNER_V4=true
        case "$mode" in
            nightly) export BUILD_TAG=5.1.x-nightly-20261020 ;;
            environment) export SCANNER_V4_FULL_BUNDLE=true ;;
            disabled) export ROX_SCANNER_V4=false ;;
        esac
        _scanner_v4_ci_helm_values "$TEST_ROOT/deploy/common/ci-values.yaml" > "$BATS_TEST_TMPDIR/rendered.yaml"
        run cmp "$TEST_ROOT/deploy/common/ci-values.yaml" "$BATS_TEST_TMPDIR/rendered.yaml"
        assert_success
    done
}

@test "standalone chart defaults to production and supports minimal versions" {
    local chart="${BATS_TEST_DIRNAME}/../../../scanner/e2etests/helmchart"
    helm template scanner "$chart" > "$BATS_TEST_TMPDIR/default.yaml"
    run grep 'name: SCANNER_V4_MATCHER_VULNERABILITY_VERSION' "$BATS_TEST_TMPDIR/default.yaml"
    assert_failure
    helm template scanner "$chart" --set-string "app.scanner.vulnerabilityVersion=$EXPECTED_VERSION" > "$BATS_TEST_TMPDIR/minimal.yaml"
    run grep "$EXPECTED_VERSION" "$BATS_TEST_TMPDIR/minimal.yaml"
    assert_success
    helm template scanner "$chart" --set-string app.scanner.vulnerabilitiesUrl=https://example.invalid/explicit.zip > "$BATS_TEST_TMPDIR/explicit.yaml"
    run grep https://example.invalid/explicit.zip "$BATS_TEST_TMPDIR/explicit.yaml"
    assert_success
}

@test "upgrade initial deployment selects minimal before startup and honors full selection" {
    source "${BATS_TEST_DIRNAME}/../../upgrade/lib.sh"
    setup_roxie_deploy_stubs
    export EARLIER_TAG=4.10.0 CI=true ROX_USERNAME=admin ROX_ADMIN_PASSWORD=test
    deploy_earlier_tag_central
    run yq eval '.central.spec.customize.envVars[] | select(.name == "SCANNER_V4_MATCHER_VULNERABILITY_VERSION") | .value' "$BATS_TEST_TMPDIR/roxie-config.yaml"
    assert_output "$EXPECTED_VERSION"
    export SCANNER_V4_FULL_BUNDLE=true
    deploy_earlier_tag_central
    run yq eval '.central.spec.customize.envVars // [] | length' "$BATS_TEST_TMPDIR/roxie-config.yaml"
    assert_output 0
}

@test "Roxie bundle selection preserves unrelated environment settings" {
    export CI=true
    printf 'central: {spec: {customize: {envVars: [{name: OTHER, value: keep}]}}}\n' > "$BATS_TEST_TMPDIR/roxie.yaml"
    _configure_roxie_ci_vuln_bundle "$BATS_TEST_TMPDIR/roxie.yaml"
    run yq eval '.central.spec.customize.envVars[] | select(.name == "OTHER") | .value' "$BATS_TEST_TMPDIR/roxie.yaml"
    assert_output keep
}

@test "Operator upgrade readiness preserves bundle and unrelated environment settings" {
    source "${BATS_TEST_DIRNAME}/../../upgrade/lib.sh"
    export CURRENT_TAG=head
    roxie() { :; }
    kubectl() {
        case " $* " in
            *" get "*) printf '{"spec":{"customize":{"envVars":[{"name":"OTHER","value":"keep"},{"name":"SCANNER_V4_MATCHER_VULNERABILITY_VERSION","value":"minimal"}]}}}' ;;
            *" patch "*) printf '%s\n' "${@: -1}" > "$BATS_TEST_TMPDIR/patch.json" ;;
        esac
    }
    wait_for_api() { :; }
    wait_for_central_db() { :; }
    wait_for_scanner_V4() { :; }
    upgrade_operator_to_head
    run jq -r '.spec.customize.envVars[] | .value' "$BATS_TEST_TMPDIR/patch.json"
    assert_output $'keep\nminimal\nvulnerability'
}

@test "local Helm defaults are pinned and full modes remove the pin" {
    source "${BATS_TEST_DIRNAME}/../../../deploy/common/k8sbased.sh"
    local values="${BATS_TEST_DIRNAME}/../../../deploy/common/local-dev-values.yaml"
    run _scanner_v4_local_helm_values "$values"
    assert_success
    assert_output --partial ci-minimal-
    for mode in environment nightly disabled; do
        unset BUILD_TAG SCANNER_V4_FULL_BUNDLE
        export ROX_SCANNER_V4=true
        case "$mode" in
            environment) export SCANNER_V4_FULL_BUNDLE=true ;;
            nightly) export BUILD_TAG=5.1.x-nightly-20261020 ;;
            disabled) export ROX_SCANNER_V4=false ;;
        esac
        run _scanner_v4_local_helm_values "$values"
        assert_success
        refute_output --partial ci-minimal-
        assert_output --partial 'memory: 4Gi'
    done
}
