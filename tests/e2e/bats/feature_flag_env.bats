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
