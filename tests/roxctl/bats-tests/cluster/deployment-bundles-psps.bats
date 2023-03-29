#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
    # remove binaries from the previous runs
    [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*
    echo "Testing roxctl version: '$(roxctl-release version)'" >&3

    [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
    [[ -n "${ROX_PASSWORD}" ]] || fail "Environment variable 'ROX_PASSWORD' required"
}

setup() {
    out_dir="$(mktemp -d -u)"
}

teardown() {
    rm -rf "$out_dir"
}

sensor_bundle_psp_enabled() {
    local cluster_type="$1"
    shift
    local sensor_name="sensor-test-${RANDOM}-${RANDOM}-${RANDOM}"
    local bundle_dir="${out_dir}/bundle-${sensor_name}"
    roxctl-release -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor generate "${cluster_type}" --name="${sensor_name}" "$@" --output-dir="${bundle_dir}"
    run grep -rq "kind: PodSecurityPolicy" "${bundle_dir}"
    assert_success
}

sensor_bundle_psp_disabled() {
    local cluster_type="$1"
    shift
    local sensor_name="sensor-test-${RANDOM}-${RANDOM}-${RANDOM}"
    local bundle_dir="${out_dir}/bundle-${sensor_name}"
    roxctl-release -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor generate "${cluster_type}" --name="${sensor_name}" "$@" --output-dir="${bundle_dir}"
    run grep -rq "kind: PodSecurityPolicy" "${bundle_dir}"
    assert_failure
}

# Testing: sensor generate k8s
@test "PodSecurityPolicies can be disabled for sensor deployment bundle (k8s)" {
    sensor_bundle_psp_disabled k8s --enable-pod-security-policies=false
}

@test "PodSecurityPolicies can be enabled for sensor deployment bundle (k8s)" {
    sensor_bundle_psp_enabled k8s --enable-pod-security-policies=true
}

@test "PodSecurityPolicies enabled by default for sensor deployment bundle (k8s)" {
    sensor_bundle_psp_enabled k8s
}

# Testing: sensor generate openshift
@test "PodSecurityPolicies can be disabled for sensor deployment bundle (openshift)" {
    sensor_bundle_psp_disabled openshift --enable-pod-security-policies=false
}

@test "PodSecurityPolicies can be enabled for sensor deployment bundle (openshift)" {
    sensor_bundle_psp_enabled openshift --enable-pod-security-policies=true
}

@test "PodSecurityPolicies enabled by default for sensor deployment bundle (openshift)" {
    sensor_bundle_psp_enabled openshift
}
