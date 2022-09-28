#!/usr/bin/env bats

load "../helpers.bash"

out_dir=""

setup_file() {
    # remove binaries from the previous runs
    [[ -n "$NO_BATS_ROXCTL_REBUILD" ]] || rm -f "${tmp_roxctl}"/roxctl*

    echo "Testing roxctl version: '$(roxctl-release version)'" >&3
    command -v yq >/dev/null || skip "Tests in this file require yq"
}

setup() {
    export out_dir="$(mktemp -d -u)"
}

teardown() {
    rm -rf "$out_dir"
}

@test "roxctl central generate k8s --enable-pod-security-policies=false should not output PSPs" {
    roxctl central generate k8s pvc --output-dir="$out_dir" --enable-pod-security-policies=false
    ! grep -rq "kind: PodSecurityPolicy" "${out_dir}/central"
}

@test "roxctl central generate k8s --enable-pod-security-policies=true should output PSPs" {
    roxctl central generate k8s pvc --output-dir="$out_dir" --enable-pod-security-policies=true
    grep -rq "kind: PodSecurityPolicy" "${out_dir}/central"
}

@test "roxctl central generate k8s should output PSPs" {
    roxctl central generate k8s pvc --output-dir="$out_dir"
    grep -rq "kind: PodSecurityPolicy" "${out_dir}/central"
}
