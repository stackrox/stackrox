#!/usr/bin/env bats

# shellcheck disable=SC1091
load "../../../scripts/test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    test_file="${BATS_TEST_TMPDIR}/roxie.yaml"
    login_calls_file="${BATS_TEST_TMPDIR}/login-calls"
    # shellcheck disable=SC2317
    registry_ro_login() { printf '%s\n' "$1" >> "$login_calls_file"; }
}

write_roxie_config() {
    local version="$1"
    local konflux_images="${2:-true}"
    cat > "$test_file" <<EOF
roxie:
  konfluxImages: $konflux_images
  version: "$version"
EOF
}

@test "prepare_for_konflux preserves release candidate tags" {
    write_roxie_config "5.0.0-rc.1"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.0.0-rc.1"
    run cat "$login_calls_file"
    assert_success
    assert_output "quay.io/rhacs-eng"
}

@test "prepare_for_konflux preserves final release tags" {
    write_roxie_config "5.0.0"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.0.0"
}

@test "prepare_for_konflux preserves release branch tags" {
    write_roxie_config "5.0.x-123-gabcdef"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.0.x-123-gabcdef"
}

@test "prepare_for_konflux preserves unsuffixed development tags" {
    write_roxie_config "5.1.x-123-gabcdef"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.1.x-123-gabcdef"
}

@test "prepare_for_konflux preserves suffixed development tags" {
    write_roxie_config "5.1.x-123-gabcdef-fast"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.1.x-123-gabcdef-fast"
}

@test "prepare_for_konflux leaves the version unchanged when Konflux images are disabled" {
    write_roxie_config "5.1.x-123-gabcdef" false

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.1.x-123-gabcdef"
    [ ! -e "$login_calls_file" ]
}
