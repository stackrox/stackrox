#!/usr/bin/env bats

# bats tests/e2e/bats/feature_flag_env.bats

# shellcheck disable=SC1091
load "../../../scripts/test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../../../deploy/common/feature-flag-env.sh"
}

@test "feature_flag_env_assignments is empty when list.go is missing" {
    run feature_flag_env_assignments "${BATS_TEST_TMPDIR}/does-not-exist.go"
    assert_success
    assert_output --partial "WARNING"
    refute_line --regexp '^ROX_'
}

@test "feature_flag_env_assignments emits only set, non-empty feature flags" {
    list_file="${BATS_TEST_TMPDIR}/list.go"
    cat > "${list_file}" <<'EOF'
	Foo = registerFeature("foo", "ROX_CISA_KEV")
	Bar = registerFeature("bar", "ROX_VULN_MGMT_LEGACY_SNOOZE")
	Baz = registerFeature("baz", "ROX_NETWORK_GRAPH_EXTERNAL_IPS")
	// URL via the ROX_SCANNER_V4_MAVEN_SEARCH_URL environment variable.
	Qux = registerFeature("qux", "ROX_SCANNER_V4_MAVEN_SEARCH")
EOF
    export ROX_CISA_KEV=true
    export ROX_NETWORK_GRAPH_EXTERNAL_IPS=false
    export ROX_SCANNER_V4_MAVEN_SEARCH_URL=http://example.invalid
    unset ROX_VULN_MGMT_LEGACY_SNOOZE || true
    export ROX_SCANNER_V4_MAVEN_SEARCH=""

    run feature_flag_env_assignments "${list_file}"
    assert_success
    assert_line "ROX_CISA_KEV=true"
    assert_line "ROX_NETWORK_GRAPH_EXTERNAL_IPS=false"
    refute_line --partial "ROX_VULN_MGMT_LEGACY_SNOOZE"
    refute_line --partial "ROX_SCANNER_V4_MAVEN_SEARCH_URL"
    refute_line --partial "ROX_SCANNER_V4_MAVEN_SEARCH="
}

@test "feature_flag_env_assignments reads pkg/features/list.go" {
    export ROX_CISA_KEV=true
    run feature_flag_env_assignments "${BATS_TEST_DIRNAME}/../../../pkg/features/list.go"
    assert_success
    assert_line "ROX_CISA_KEV=true"
}

@test "is_helm_owned_feature_flag covers Central chart first-class scanner env" {
    run is_helm_owned_feature_flag ROX_SCANNER_V4
    assert_success
    run is_helm_owned_feature_flag ROX_LEGACY_SCANNER
    assert_success
    run is_helm_owned_feature_flag ROX_CISA_KEV
    assert_failure
    run is_helm_owned_feature_flag ROX_VIRTUAL_MACHINES
    assert_failure
}

@test "omit_helm_owned_feature_flags drops chart-owned names and keeps the rest" {
    run omit_helm_owned_feature_flags <<'EOF'
ROX_CISA_KEV=true
ROX_SCANNER_V4=true
ROX_LEGACY_SCANNER=false
ROX_NETWORK_GRAPH_EXTERNAL_IPS=false
EOF
    assert_success
    assert_line "ROX_CISA_KEV=true"
    assert_line "ROX_NETWORK_GRAPH_EXTERNAL_IPS=false"
    refute_line --partial "ROX_SCANNER_V4"
    refute_line --partial "ROX_LEGACY_SCANNER"
}

@test "feature_flag_env_assignments still emits helm-owned flags for kubectl inject" {
    list_file="${BATS_TEST_TMPDIR}/list.go"
    cat > "${list_file}" <<'EOF'
	ScannerV4 = registerFeature("scanner v4", "ROX_SCANNER_V4")
	Legacy = registerFeature("legacy scanner", "ROX_LEGACY_SCANNER")
	Foo = registerFeature("foo", "ROX_CISA_KEV")
EOF
    export ROX_SCANNER_V4=true
    export ROX_LEGACY_SCANNER=false
    export ROX_CISA_KEV=true

    run feature_flag_env_assignments "${list_file}"
    assert_success
    assert_line "ROX_SCANNER_V4=true"
    assert_line "ROX_LEGACY_SCANNER=false"
    assert_line "ROX_CISA_KEV=true"

    run omit_helm_owned_feature_flags < <(feature_flag_env_assignments "${list_file}")
    assert_success
    assert_line "ROX_CISA_KEV=true"
    refute_line --partial "ROX_SCANNER_V4"
    refute_line --partial "ROX_LEGACY_SCANNER"
}
