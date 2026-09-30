#!/usr/bin/env bats

# shellcheck disable=SC1091
load "../../../scripts/test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    test_file="${BATS_TEST_TMPDIR}/roxie.yaml"
    # shellcheck disable=SC2317
    registry_ro_login() { :; }
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

set_github_ci_ref() {
    export CI=true
    export GITHUB_ACTION=true
    unset GITHUB_HEAD_REF
    export GITHUB_REF_NAME="$1"
}

@test "prepare_for_konflux preserves release candidate tags" {
    write_roxie_config "5.0.0-rc.1"
    set_github_ci_ref "5.0.0-rc.1"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.0.0-rc.1"
}

@test "prepare_for_konflux preserves final release tags" {
    write_roxie_config "5.0.0"
    set_github_ci_ref "5.0.0"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.0.0"
}

@test "prepare_for_konflux preserves tags on release branches" {
    write_roxie_config "5.0.x-123-gabcdef"
    set_github_ci_ref "release-5.0"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.0.x-123-gabcdef"
}

@test "prepare_for_konflux adds the development suffix on non-release branches" {
    write_roxie_config "5.1.x-123-gabcdef"
    set_github_ci_ref "master"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.1.x-123-gabcdef-fast"
}

@test "prepare_for_konflux preserves an existing development suffix" {
    write_roxie_config "5.1.x-123-gabcdef-fast"
    set_github_ci_ref "master"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.1.x-123-gabcdef-fast"
}

@test "prepare_for_konflux leaves the version unchanged when Konflux images are disabled" {
    write_roxie_config "5.1.x-123-gabcdef" false
    set_github_ci_ref "master"

    prepare_for_konflux "$test_file"

    run yq eval ".roxie.version" "$test_file"
    assert_success
    assert_output "5.1.x-123-gabcdef"
}
