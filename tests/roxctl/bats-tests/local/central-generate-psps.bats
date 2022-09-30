#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
    # remove binaries from the previous runs
    [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*
    echo "Testing roxctl version: '$(roxctl-release version)'" >&3

    command -v grep || skip "Command 'grep' required."
    [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
    [[ -n "${ROX_PASSWORD}" ]] || fail "Environment variable 'ROX_PASSWORD' required"
}

setup() {
    export out_dir="$(mktemp -d -u)"
}

teardown() {
    rm -rf "$out_dir"
}

# Testing: central generate k8s

@test "roxctl-release central generate k8s --enable-pod-security-policies=false should not output PSPs" {
    roxctl-release central generate k8s pvc --output-dir="$out_dir" --enable-pod-security-policies=false
    run grep -rq "kind: PodSecurityPolicy" "${out_dir}/central"
    assert_failure
}

@test "roxctl-release central generate k8s --enable-pod-security-policies=true should output PSPs" {
    roxctl-release central generate k8s pvc --output-dir="$out_dir" --enable-pod-security-policies=true
    run grep -rq "kind: PodSecurityPolicy" "${out_dir}/central"
    assert_success
}

@test "roxctl-release central generate k8s should output PSPs" {
    roxctl-release central generate k8s pvc --output-dir="$out_dir"
    run grep -rq "kind: PodSecurityPolicy" "${out_dir}/central"
    assert_success
}

# Testing: sensor generate k8s
@test "sensor" {
    sensor_name="sensor-test-${RANDOM}"
    roxctl-release -e "$API_ENDPOINT" -p "$ROX_PASSWORD" sensor generate k8s --name="${sensor_name}" --output-dir="${out_dir}/${sensor_name}"
}
