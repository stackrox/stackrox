#!/usr/bin/env bats

load "../helpers.bash"

setup_file() {
    local -r roxctl_version="$(roxctl-development version || true)"
    echo "Testing roxctl version: '${roxctl_version}'" >&3

    command -v curl || skip "Command 'curl' required."
    [[ -n "${API_ENDPOINT}" ]] || fail "Environment variable 'API_ENDPOINT' required"
    [[ -n "${ROX_ADMIN_PASSWORD}" ]] || fail "Environment variable 'ROX_ADMIN_PASSWORD' required"

    export crs_name="crs-${RANDOM}-${RANDOM}"
    out_dir="$(mktemp -d)"; export out_dir
    export ROX_CLUSTER_REGISTRATION_SECRETS=true
}

teardown_file() {
    if [ -n "${out_dir:-}" ]; then
        rm -rf "${out_dir}"
    fi
}

@test "CRS can be issued" {
    local crs_file="${out_dir}/${crs_name}.yaml"
    roxctl_authenticated central crs generate "${crs_name}" -o "${crs_file}"
    test -f "${crs_file}"
}
@test "CRS is listed" {
    grep "^${crs_name}[[:space:]]" <(roxctl_authenticated central crs list)
}
@test "CRS can be revoked" {
    roxctl_authenticated central crs revoke "${crs_name}"
}

@test "CRS is not listed anymore after revoking" {
    run grep "^${crs_name}[[:space:]]" <(roxctl_authenticated central crs list)
    assert_failure
}

@test "CRS generation fails if output file already exists" {
    local crs_file="${out_dir}/${crs_name}.yaml"
    local content="foo"
    echo -n "${content}" > "${crs_file}"
    run roxctl_authenticated central crs generate "${crs_name}" -o "${crs_file}"
    assert_failure
    [[ -f "${crs_file}" ]]
    [[ "$(cat "${crs_file}")" == "${content}" ]]
}

@test "Revoking non-existant CRS fails" {
    local crs_file="${out_dir}/${crs_name}.yaml"
    run roxctl_authenticated central crs revoke "i-dont-exist"
    assert_failure
}
