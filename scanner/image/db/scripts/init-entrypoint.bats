#!/usr/bin/env bats

# Tests for init-entrypoint.sh downgrade detection logic.

bats_helpers_root=${BATS_CORE_ROOT:-${HOME}/bats-core}
if [[ ! -f "${bats_helpers_root}/bats-support/load.bash" ]]; then
  bats_helpers_root="/usr/lib/node_modules"
fi
load "${bats_helpers_root}/bats-support/load.bash"
load "${bats_helpers_root}/bats-assert/load.bash"

setup() {
    source "${BATS_TEST_DIRNAME}/init-entrypoint.sh"
}

# -- extract_major_minor tests --

@test "extract_major_minor: release version" {
    run extract_major_minor "4.8.1"
    assert_success
    assert_output "4.8"
}

@test "extract_major_minor: release candidate" {
    run extract_major_minor "5.1.0-rc.2"
    assert_success
    assert_output "5.1"
}

@test "extract_major_minor: dev build" {
    run extract_major_minor "5.2.x-23-gabcdef"
    assert_success
    assert_output "5.2"
}

@test "extract_major_minor: major only returns empty" {
    run extract_major_minor "5"
    assert_success
    assert_output ""
}

@test "extract_major_minor: empty string returns empty" {
    run extract_major_minor ""
    assert_success
    assert_output ""
}

@test "extract_major_minor: garbage returns empty" {
    run extract_major_minor "not-a-version"
    assert_success
    assert_output ""
}

@test "extract_major_minor: two-digit components" {
    run extract_major_minor "12.34.56"
    assert_success
    assert_output "12.34"
}

# -- is_downgrade tests --

@test "is_downgrade: 5.1 -> 4.8 is a downgrade" {
    run is_downgrade "5.1" "4.8"
    assert_success
}

@test "is_downgrade: 4.8 -> 5.1 is NOT a downgrade" {
    run is_downgrade "4.8" "5.1"
    assert_failure
}

@test "is_downgrade: same version is NOT a downgrade" {
    run is_downgrade "5.1" "5.1"
    assert_failure
}

@test "is_downgrade: 5.1 -> 5.0 is a downgrade" {
    run is_downgrade "5.1" "5.0"
    assert_success
}

@test "is_downgrade: 5.0 -> 5.1 is NOT a downgrade" {
    run is_downgrade "5.0" "5.1"
    assert_failure
}

@test "is_downgrade: empty stored is NOT a downgrade" {
    run is_downgrade "" "5.1"
    assert_failure
}

@test "is_downgrade: empty current is NOT a downgrade" {
    run is_downgrade "5.1" ""
    assert_failure
}

@test "is_downgrade: both empty is NOT a downgrade" {
    run is_downgrade "" ""
    assert_failure
}

@test "is_downgrade: major version decrease 5.0 -> 4.9" {
    run is_downgrade "5.0" "4.9"
    assert_success
}

@test "is_downgrade: 10.1 -> 9.99 is a downgrade" {
    run is_downgrade "10.1" "9.99"
    assert_success
}

# -- main() integration tests --

setup_pgdata() {
    export PGDATA
    PGDATA=$(mktemp -d)/pgdata
    mkdir -p "$PGDATA"
}

teardown_pgdata() {
    rm -rf "$(dirname "$PGDATA")"
}

@test "main: downgrade wipes init-ready and reinitializes" {
    setup_pgdata

    local version_file="$PGDATA/.scanner-db-version"

    # Simulate an existing initialized DB at version 5.1
    touch "$PGDATA/.init-ready"
    echo "5.1" > "$version_file"

    # Override destructive commands with stubs
    initdb() { mkdir -p "$PGDATA"; touch "$PGDATA/.initdb-ran"; }
    export -f initdb

    export SCANNER_V4_DB_VERSION="4.8.0"
    export POSTGRES_HOST_AUTH_METHOD="scram-sha-256"
    export POSTGRES_PASSWORD_FILE="/dev/null"

    main

    # .init-ready should be recreated by the init flow
    assert [ -e "$PGDATA/.init-ready" ]
    # initdb should have run (DB was wiped)
    assert [ -e "$PGDATA/.initdb-ran" ]
    # Version file should reflect the downgraded version
    assert_equal "$(cat "$version_file")" "4.8"

    teardown_pgdata
}

@test "main: upgrade does NOT wipe the database" {
    setup_pgdata

    local version_file="$PGDATA/.scanner-db-version"

    # Simulate an existing initialized DB at version 4.8
    touch "$PGDATA/.init-ready"
    echo "4.8" > "$version_file"

    export SCANNER_V4_DB_VERSION="5.1.0"

    main

    # .init-ready should still be there (not removed and recreated)
    assert [ -e "$PGDATA/.init-ready" ]
    # Version file should be updated
    assert_equal "$(cat "$version_file")" "5.1"

    teardown_pgdata
}

@test "main: same version does NOT wipe the database" {
    setup_pgdata

    local version_file="$PGDATA/.scanner-db-version"

    touch "$PGDATA/.init-ready"
    echo "5.1" > "$version_file"
    # Create a marker to verify PGDATA is not wiped
    touch "$PGDATA/should-survive"

    export SCANNER_V4_DB_VERSION="5.1.2"

    main

    assert [ -e "$PGDATA/.init-ready" ]
    assert [ -e "$PGDATA/should-survive" ]
    assert_equal "$(cat "$version_file")" "5.1"

    teardown_pgdata
}

@test "main: no version file (legacy install) does NOT wipe" {
    setup_pgdata

    touch "$PGDATA/.init-ready"
    touch "$PGDATA/should-survive"

    export SCANNER_V4_DB_VERSION="5.1.0"

    main

    assert [ -e "$PGDATA/.init-ready" ]
    assert [ -e "$PGDATA/should-survive" ]
    # Version file should now exist
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.1"

    teardown_pgdata
}

@test "main: no SCANNER_V4_DB_VERSION skips version tracking" {
    setup_pgdata

    touch "$PGDATA/.init-ready"

    unset SCANNER_V4_DB_VERSION

    main

    assert [ -e "$PGDATA/.init-ready" ]
    # No version file should be created
    assert [ ! -e "$PGDATA/.scanner-db-version" ]

    teardown_pgdata
}

@test "main: malformed stored version warns and reinitializes" {
    setup_pgdata

    local version_file="$PGDATA/.scanner-db-version"

    touch "$PGDATA/.init-ready"
    echo "garbage" > "$version_file"

    initdb() { mkdir -p "$PGDATA"; touch "$PGDATA/.initdb-ran"; }
    export -f initdb

    export SCANNER_V4_DB_VERSION="5.1.0"
    export POSTGRES_HOST_AUTH_METHOD="scram-sha-256"
    export POSTGRES_PASSWORD_FILE="/dev/null"

    run main

    assert_success
    assert_output --partial "malformed version"
    # initdb should have run (DB was reinitialized)
    assert [ -e "$PGDATA/.initdb-ran" ]
    # .init-ready should be recreated by the init flow
    assert [ -e "$PGDATA/.init-ready" ]
    # Version file should reflect the current version
    assert_equal "$(cat "$version_file")" "5.1"

    teardown_pgdata
}

@test "main: empty stored version warns and reinitializes" {
    setup_pgdata

    local version_file="$PGDATA/.scanner-db-version"

    touch "$PGDATA/.init-ready"
    : > "$version_file"

    initdb() { mkdir -p "$PGDATA"; touch "$PGDATA/.initdb-ran"; }
    export -f initdb

    export SCANNER_V4_DB_VERSION="5.1.0"
    export POSTGRES_HOST_AUTH_METHOD="scram-sha-256"
    export POSTGRES_PASSWORD_FILE="/dev/null"

    run main

    assert_success
    assert_output --partial "malformed version"
    assert [ -e "$PGDATA/.initdb-ran" ]
    assert [ -e "$PGDATA/.init-ready" ]
    assert_equal "$(cat "$version_file")" "5.1"

    teardown_pgdata
}

@test "main: patch rollback within same minor does NOT wipe" {
    setup_pgdata

    local version_file="$PGDATA/.scanner-db-version"

    touch "$PGDATA/.init-ready"
    echo "5.1" > "$version_file"
    touch "$PGDATA/should-survive"

    # 5.1.2 -> 5.1.1 is a patch rollback; major.minor is the same (5.1)
    export SCANNER_V4_DB_VERSION="5.1.1"

    main

    assert [ -e "$PGDATA/.init-ready" ]
    assert [ -e "$PGDATA/should-survive" ]

    teardown_pgdata
}
