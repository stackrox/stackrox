#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
    # remove binaries from the previous runs
    [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*
    echo "Testing roxctl version: '$(roxctl-release version)'" >&3
}

setup() {
    out_dir="$(mktemp -d -u)"
}

teardown() {
    rm -rf "$out_dir"
}

central_bundle_psp_enabled() {
    local cluster_type="$1"
    shift
    local bundle_dir="${out_dir}/bundle-central-${RANDOM}-${RANDOM}-${RANDOM}"
    roxctl-release central generate "${cluster_type}" pvc "$@" --output-dir="${bundle_dir}"
    run grep -rq "kind: PodSecurityPolicy" "${bundle_dir}"
    assert_success
}

central_bundle_psp_disabled() {
    local cluster_type="$1"
    shift
    local bundle_dir="${out_dir}/bundle-central-${RANDOM}-${RANDOM}-${RANDOM}"
    roxctl-release central generate "${cluster_type}" pvc "$@" --output-dir="${bundle_dir}"
    run grep -rq "kind: PodSecurityPolicy" "${bundle_dir}"
    assert_failure
}

# Testing: central generate k8s
@test "PodSecurityPolicies can be disabled for central deployment bundle (k8s)" {
    central_bundle_psp_enabled k8s --enable-pod-security-policies=false
}

@test "PodSecurityPolicies can be enabled for central deployment bundle (k8s)" {
    central_bundle_psp_enabled k8s --enable-pod-security-policies=true
}

@test "PodSecurityPolicies are enabled by default for central deployment bundle (k8s)" {
    central_bundle_psp_enabled k8s
}

# Testing: central generate openshift
@test "PodSecurityPolicies can be disabled for central deployment bundle (openshift)" {
    central_bundle_psp_enabled openshift --enable-pod-security-policies=false
}

@test "PodSecurityPolicies can be enabled for central deployment bundle (openshift)" {
    central_bundle_psp_enabled openshift --enable-pod-security-policies=true
}

@test "PodSecurityPolicies are enabled by default for central deployment bundle (openshift)" {
    central_bundle_psp_enabled openshift
}
