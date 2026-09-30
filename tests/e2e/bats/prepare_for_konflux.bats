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
    local konflux_images="$1"
    cat > "$test_file" <<EOF
roxie:
  konfluxImages: $konflux_images
  version: "5.1.x-123-gabcdef-fast"
EOF
}

@test "prepare_for_konflux logs in to the Konflux registry when Konflux images are enabled" {
    write_roxie_config true

    prepare_for_konflux "$test_file"

    run cat "$login_calls_file"
    assert_success
    assert_output "quay.io/rhacs-eng"
}

@test "prepare_for_konflux does not log in when Konflux images are disabled" {
    write_roxie_config false

    prepare_for_konflux "$test_file"

    [ ! -e "$login_calls_file" ]
}
