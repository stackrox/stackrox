#!/usr/bin/env bats

# Unit tests for the StackRox additions to the scanner-v4-db entrypoint,
# scanner/image/db/scripts/docker-entrypoint.sh.
#
# Run with:
#
#   bats scanner/image/db/bats/docker_entrypoint.bats

load "../../../../scripts/test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../scripts/docker-entrypoint.sh"

    export PGDATA="${BATS_TEST_TMPDIR}/pgdata"
    mkdir -p "$PGDATA"

    # In place of /usr/lib64/pgsql in the image.
    prev_dir="${BATS_TEST_TMPDIR}/pgsql"

    image_bin="${BATS_TEST_TMPDIR}/bin"
    mkdir -p "$image_bin"
    export PATH="${image_bin}:${PATH}"
}

# given_binary_version makes `postgres -V` report the given major version.
function given_binary_version() {
    local major="$1"
    mock_executable "${image_bin}/postgres" \
        "echo 'postgres (PostgreSQL) ${major}.2'"
}

# given_data_version writes $PGDATA/PG_VERSION the way initdb would.
function given_data_version() {
    local major="$1"
    echo "$major" > "${PGDATA}/PG_VERSION"
}

# given_prev_binaries mimics the postgres executable under /usr/lib64/pgsql.
function given_prev_binaries() {
    local major="$1"
    mock_executable "${prev_dir}/postgresql-${major}/bin/postgres" \
        "echo 'PostgreSQL ${major} server'"
}

function mock_executable() {
    local target_path="$1"
    local script_body="$2"

    mkdir -p "$(dirname "$target_path")"
    printf '#!/usr/bin/env bash\n%s\n' "$script_body" > "$target_path"
    chmod +x "$target_path"
}

# Calls rox_use_matching_binaries and gathers results.
function use_matching_binaries() {
    local status=0

    rox_use_matching_binaries "$prev_dir" || status=$?
    printf 'first-path=%s\n' "${PATH%%:*}"
    printf 'path=%s\n' "$PATH"
    return "$status"
}

# == rox_use_matching_binaries ==
#
# The function must always return 0: under `set -e` a non-zero return takes the
# container down.

@test "rox_use_matching_binaries - fresh install, no PG_VERSION yet" {
    run use_matching_binaries
    assert_success
    assert_line "first-path=${image_bin}"
}

@test "rox_use_matching_binaries - normal boot" {
    given_binary_version 16
    given_data_version 16
    run use_matching_binaries
    assert_success
    refute_output --partial "no executable"
    assert_line "first-path=${image_bin}"
}

@test "rox_use_matching_binaries - data are newer than the image" {
    # Unsupported. The image has no newer binaries.
    given_binary_version 16
    given_data_version 17
    run use_matching_binaries
    assert_success
    refute_output --partial "no executable"
    assert_line "first-path=${image_bin}"
}

@test "rox_use_matching_binaries - data match the previous binaries" {
    local data_major=15
    given_binary_version 17
    given_data_version "$data_major"
    given_prev_binaries "$data_major"
    local before="$PATH"
    run use_matching_binaries
    assert_success
    assert_output --partial "Data are still in PostgreSQL ${data_major}, starting the binaries in ${prev_dir}/postgresql-${data_major}/bin"
    assert_line "path=${prev_dir}/postgresql-${data_major}/bin:${before}"
}

@test "rox_use_matching_binaries - data do not match the previous binaries" {
    local data_major=14
    given_binary_version 16
    given_prev_binaries 15
    given_data_version "$data_major"
    run use_matching_binaries
    assert_success
    assert_output --partial "Data are in PostgreSQL ${data_major}, but there is no executable ${prev_dir}/postgresql-${data_major}/bin/postgres"
    assert_line "first-path=${image_bin}"
}

@test "rox_use_matching_binaries - matching postgres is not executable" {
    given_binary_version 16
    given_data_version 15
    given_prev_binaries 15
    chmod -x "${prev_dir}/postgresql-15/bin/postgres"
    run use_matching_binaries
    assert_success
    assert_output --partial "no executable"
    assert_line "first-path=${image_bin}"
}

@test "rox_use_matching_binaries - corrupted PG_VERSION" {
    given_data_version "not-a-version"
    run use_matching_binaries
    assert_success
    assert_output --partial "Cannot read a valid major version from ${PGDATA}/PG_VERSION"
    assert_line "first-path=${image_bin}"
}

@test "rox_use_matching_binaries - unparsable binary version" {
    mock_executable "${image_bin}/postgres" "echo 'something else entirely'"
    given_data_version 15
    given_prev_binaries 15
    run use_matching_binaries
    assert_success
    assert_output --partial "Cannot read a valid major version from postgres -V"
    assert_line "first-path=${image_bin}"
}

@test "rox_use_matching_binaries - postgres -V failure does not crash the container" {
    mock_executable "${image_bin}/postgres" "exit 127"
    given_data_version 15
    given_prev_binaries 15
    # Called without `run`, which would swallow an abort.
    rox_use_matching_binaries "$prev_dir"
    assert_equal "${PATH%%:*}" "$image_bin"
}
