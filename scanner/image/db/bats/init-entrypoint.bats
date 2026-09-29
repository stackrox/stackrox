#!/usr/bin/env bats

load "../../../../scripts/test_helpers.bats"
load helpers

setup() {
    export TEST_DIR="$(mktemp -d)"
    export PGDATA="$TEST_DIR/data"
    export NEW_BINARIES="$TEST_DIR/bin"
    mkdir -p "$NEW_BINARIES"
    mkdir -p "$TEST_DIR/pgsql/postgresql-15/bin"

    export POSTGRES_HOST_AUTH_METHOD="scram-sha-256"
    export POSTGRES_PASSWORD_FILE="$TEST_DIR/password"
    echo "testpass" > "$POSTGRES_PASSWORD_FILE"

    export PATH="$NEW_BINARIES:$PATH"

    source "${BATS_TEST_DIRNAME}/../scripts/init-entrypoint.sh"
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Test helper functions
init_pgdata() {
    mkdir -p "$PGDATA"
    touch "$PGDATA/.init-ready"
}

setup_binaries() {
    local old_version=$1 new_version=$2 pg_upgrade_mode=${3:-succeed}
    local old_bin="$TEST_DIR/pgsql/postgresql-${old_version}/bin"

    given_binary_version "$NEW_BINARIES" "$new_version"
    mock_executable "$old_bin/pg_ctl" ''
    mock_executable "$old_bin/pg_isready" ''
    mock_executable "$old_bin/pg_controldata" "echo 'Database cluster state:               shutdown'"
    given_initdb "$NEW_BINARIES" "$new_version"

    case "$pg_upgrade_mode" in
        fail)
            mock_executable "$NEW_BINARIES/pg_upgrade" 'exit 1'
            ;;
        fail-clone-only)
            mock_executable "$NEW_BINARIES/pg_upgrade" '[[ "$*" == *"--clone"* ]] && exit 1; exit 0'
            ;;
        fail-actual-only)
            mock_executable "$NEW_BINARIES/pg_upgrade" '[[ "$*" == *"--check"* ]] && exit 0; exit 1'
            ;;
        *)
            mock_executable "$NEW_BINARIES/pg_upgrade" ''
            ;;
    esac
}

# given_pending_statistics prepares a cluster that owes a statistics refresh,
# at a version the upgrade path will not touch. Each mock leaves a file behind
# so tests can tell what ran; vacuumdb only does so for the expected flags,
# and otherwise fails the way a real one would.
given_pending_statistics() {
    init_pgdata
    given_data_version "$PGDATA" 16
    given_binary_version "$NEW_BINARIES" 16
    mock_executable "$NEW_BINARIES/pg_ctl" 'touch "$TEST_DIR/started"'
    mock_executable "$NEW_BINARIES/vacuumdb" \
        '[[ "$*" == *--all* && "$*" == *--analyze-only* ]] && touch "$TEST_DIR/analyzed"'
    touch "$PGDATA/.stats-pending"
}

assert_upgrade() {
    local expected_version=$1
    [ -f "$PGDATA/.init-ready" ] || return 1
    [[ "$(cat "$PGDATA/PG_VERSION")" == "$expected_version" ]] || return 1
    [ ! -d "$PGDATA-new" ] || return 1
    [ ! -d "$PGDATA-old" ] || return 1
}

# Version downgrade detection
@test "extract-major-minor: release version" {
    run extract-major-minor "4.8.1"
    assert_success
    assert_output "4.8"
}

@test "extract-major-minor: release candidate" {
    run extract-major-minor "5.1.0-rc.2"
    assert_success
    assert_output "5.1"
}

@test "extract-major-minor: dev build" {
    run extract-major-minor "5.2.x-23-gabcdef"
    assert_success
    assert_output "5.2"
}

@test "extract-major-minor: major only returns empty" {
    run extract-major-minor "5"
    assert_success
    assert_output ""
}

@test "extract-major-minor: empty string returns empty" {
    run extract-major-minor ""
    assert_success
    assert_output ""
}

@test "extract-major-minor: garbage returns empty" {
    run extract-major-minor "not-a-version"
    assert_success
    assert_output ""
}

@test "extract-major-minor: two-digit components" {
    run extract-major-minor "12.34.56"
    assert_success
    assert_output "12.34"
}

@test "is-downgrade: 5.1 -> 4.8 is a downgrade" {
    run is-downgrade "5.1" "4.8"
    assert_success
}

@test "is-downgrade: 4.8 -> 5.1 is NOT a downgrade" {
    run is-downgrade "4.8" "5.1"
    assert_failure
}

@test "is-downgrade: same version is NOT a downgrade" {
    run is-downgrade "5.1" "5.1"
    assert_failure
}

@test "is-downgrade: 5.1 -> 5.0 is a downgrade" {
    run is-downgrade "5.1" "5.0"
    assert_success
}

@test "is-downgrade: 5.0 -> 5.1 is NOT a downgrade" {
    run is-downgrade "5.0" "5.1"
    assert_failure
}

@test "is-downgrade: empty stored is NOT a downgrade" {
    run is-downgrade "" "5.1"
    assert_failure
}

@test "is-downgrade: empty current is NOT a downgrade" {
    run is-downgrade "5.1" ""
    assert_failure
}

@test "is-downgrade: both empty is NOT a downgrade" {
    run is-downgrade "" ""
    assert_failure
}

@test "is-downgrade: major version decrease 5.0 -> 4.9" {
    run is-downgrade "5.0" "4.9"
    assert_success
}

@test "is-downgrade: 10.1 -> 9.99 is a downgrade" {
    run is-downgrade "10.1" "9.99"
    assert_success
}

# ACS version tracking
@test "main: upgrade updates the ACS version without wiping the database" {
    init_pgdata
    local version_file="$PGDATA/.scanner-db-version"
    echo "4.8" > "$version_file"
    touch "$PGDATA/should-survive"
    export BUILD_TAG="5.1.0"

    run _main

    assert_success
    [ -e "$PGDATA/.init-ready" ]
    [ -e "$PGDATA/should-survive" ]
    assert_equal "$(cat "$version_file")" "5.1"
}

@test "main: same version does NOT wipe the database" {
    init_pgdata
    local version_file="$PGDATA/.scanner-db-version"
    echo "5.1" > "$version_file"
    touch "$PGDATA/should-survive"
    export BUILD_TAG="5.1.2"

    run _main

    assert_success
    [ -e "$PGDATA/.init-ready" ]
    [ -e "$PGDATA/should-survive" ]
    assert_equal "$(cat "$version_file")" "5.1"
}

@test "main: no version file (legacy install) does NOT wipe" {
    init_pgdata
    touch "$PGDATA/should-survive"
    export BUILD_TAG="5.1.0"

    run _main

    assert_success
    [ -e "$PGDATA/.init-ready" ]
    [ -e "$PGDATA/should-survive" ]
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.1"
}

@test "main: no BUILD_TAG skips version tracking" {
    init_pgdata
    unset BUILD_TAG

    run _main

    assert_success
    [ -e "$PGDATA/.init-ready" ]
    [ ! -e "$PGDATA/.scanner-db-version" ]
}

@test "main: latest skips downgrade detection and preserves the database" {
    init_pgdata
    local version_file="$PGDATA/.scanner-db-version"
    echo "5.1" > "$version_file"
    touch "$PGDATA/should-survive"
    export BUILD_TAG="latest"

    run _main

    assert_success
    [ -e "$PGDATA/.init-ready" ]
    [ -e "$PGDATA/should-survive" ]
    assert_equal "$(cat "$version_file")" "5.1"
}

@test "main: malformed stored version warns and reinitializes" {
    init_pgdata
    local version_file="$PGDATA/.scanner-db-version"
    echo "garbage" > "$version_file"
    given_initdb "$NEW_BINARIES" 16
    export BUILD_TAG="5.1.0"

    run _main

    assert_success
    assert_output --partial "malformed version"
    assert_upgrade "16"
    assert_equal "$(cat "$version_file")" "5.1"
}

@test "main: empty stored version warns and reinitializes" {
    init_pgdata
    local version_file="$PGDATA/.scanner-db-version"
    : > "$version_file"
    given_initdb "$NEW_BINARIES" 16
    export BUILD_TAG="5.1.0"

    run _main

    assert_success
    assert_output --partial "malformed version"
    assert_upgrade "16"
    assert_equal "$(cat "$version_file")" "5.1"
}

@test "main: patch rollback within the same minor does NOT wipe" {
    init_pgdata
    echo "5.1" > "$PGDATA/.scanner-db-version"
    touch "$PGDATA/should-survive"
    export BUILD_TAG="5.1.1"

    run _main

    assert_success
    [ -e "$PGDATA/.init-ready" ]
    [ -e "$PGDATA/should-survive" ]
}

# Fresh initialization
@test "fresh init creates database" {
    given_initdb "$NEW_BINARIES" 16

    init-cluster "$PGDATA"
    [[ "$(cat "$PGDATA/PG_VERSION")" == 16 ]]
}

# Normal boot
@test "upgrade - data version matches binary version" {
    init_pgdata
    given_data_version "$PGDATA" 16
    given_binary_version "$TEST_DIR/bin" 16
    export PATH="$TEST_DIR/bin:$PATH"

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"no upgrade needed"* ]]
    assert_upgrade "16"
}

# Upgrades
@test "upgrade - no old binaries" {
    init_pgdata
    given_data_version "$PGDATA" 15
    given_binary_version "$TEST_DIR/bin" 16
    export PATH="$TEST_DIR/bin:$PATH"

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"No PostgreSQL 15 binaries"* ]]
    assert_upgrade "15"
}

@test "upgrade - corrupt PG_VERSION" {
    init_pgdata
    given_data_version "$PGDATA" not-a-version
    given_binary_version "$TEST_DIR/bin" 16
    export PATH="$TEST_DIR/bin:$PATH"

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"Cannot read a valid major version"* ]]
    assert_upgrade "not-a-version"
}

@test "upgrade - postgres -V fails" {
    init_pgdata
    given_data_version "$PGDATA" 15
    mkdir -p "$TEST_DIR/bin"
    mock_executable "$TEST_DIR/bin/postgres" 'exit 1'
    export PATH="$TEST_DIR/bin:$PATH"

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"Cannot read a valid major version from postgres -V"* ]]
    assert_upgrade "15"
}

@test "upgrade - cannot clean shutdown" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    mock_executable "$TEST_DIR/pgsql/postgresql-15/bin/pg_ctl" 'exit 1'

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"not shut down cleanly"* ]]
    assert_upgrade "15"
}

@test "upgrade - init-cluster fails" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    mock_executable "$TEST_DIR/bin/initdb" 'exit 1'

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"Could not initialize"* ]]
    assert_upgrade "15"
}

@test "upgrade - actual pg_upgrade fails" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16 fail-actual-only

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"Upgrade failed"* ]]
    assert_upgrade "15"
}

@test "upgrade - clone mode succeeds" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    assert_upgrade "16"
}

@test "upgrade - copy mode succeeds" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16 fail-clone-only
    given_available_space "$NEW_BINARIES" 1000 1000000

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    assert_upgrade "16"
}

@test "upgrade - clone fails, no space for copy" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16 fail-clone-only
    given_available_space "$TEST_DIR/bin" 1000000 10000

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"Not enough free space"* ]]
    assert_upgrade "15"
}

@test "upgrade - no transfer mode available" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16 fail

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    [[ "$output" == *"No usable transfer mode"* ]]
    assert_upgrade "15"
}

@test "upgrade - preserves ACS version when current version is not provided" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    echo "5.1" > "$PGDATA/.scanner-db-version"

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    assert_upgrade "16"
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.1"
}

@test "upgrade - writes current ACS version before marking the new cluster ready" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    echo "5.1" > "$PGDATA/.scanner-db-version"

    run maybe-upgrade "$TEST_DIR/pgsql" "5.2"
    assert_success
    assert_upgrade "16"
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.2"
}

@test "main - updates ACS version without a PostgreSQL upgrade" {
    init_pgdata
    given_data_version "$PGDATA" 16
    given_binary_version "$NEW_BINARIES" 16
    echo "5.1" > "$PGDATA/.scanner-db-version"
    export BUILD_TAG="5.2.0"

    run _main
    assert_success
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.2"
}

@test "ACS version marker is not rewritten when its value is unchanged" {
    mkdir -p "$PGDATA"
    echo "5.2" > "$PGDATA/.scanner-db-version"
    mv() { return 1; }

    run set-acs-version "$PGDATA" "5.2"
    assert_success
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.2"
}

@test "ACS version marker temp files from an interrupted write are cleaned up" {
    mkdir -p "$PGDATA"
    local stale_version_file
    stale_version_file=$(mktemp "$PGDATA/.scanner-db-version.XXXXXX")

    run set-acs-version "$PGDATA" ""
    assert_success
    [ ! -e "$stale_version_file" ]
}

@test "ACS version marker update continues when stale temp cleanup fails" {
    mkdir -p "$PGDATA"
    rm() {
        if [ "$1" = "-f" ]; then
            return 1
        fi
        command rm "$@"
    }

    run set-acs-version "$PGDATA" "5.2"
    assert_success
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.2"
}

@test "upgrade - marker copy failure does not fail the upgrade" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    echo "5.1" > "$PGDATA/.scanner-db-version"
    cp() { return 1; }

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    assert_output --partial "continuing without the marker"
    assert_upgrade "16"
    [ ! -e "$PGDATA/.scanner-db-version" ]
    [ ! -d "$PGDATA-old" ]
    [ ! -e "$PGDATA-new/.init-ready" ]
}

@test "upgrade - interrupted swap restores ACS version for downgrade detection" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    echo "5.1" > "$PGDATA/.scanner-db-version"
    mv() {
        [ "$1" != "$NEW_DATADIR" ] || return 1
        command mv "$@"
    }

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_failure
    [ ! -d "$PGDATA" ]
    assert_equal "$(cat "$PGDATA-new/.scanner-db-version")" "5.1"

    recover-interrupted-upgrade
    assert_upgrade "15"
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.1"

    export BUILD_TAG="5.0.0"
    run _main
    assert_success
    assert_output --partial "downgrade detected"
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.0"
    assert_upgrade "16"
}

@test "main - ACS downgrade skips PostgreSQL upgrade and statistics refresh" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    echo "5.1" > "$PGDATA/.scanner-db-version"
    touch "$PGDATA/.stats-pending" "$PGDATA/should-be-removed"
    export BUILD_TAG="5.0.0"
    maybe-upgrade() { touch "$TEST_DIR/upgrade-ran"; }
    maybe-refresh-statistics() { touch "$TEST_DIR/refresh-ran"; }

    run _main
    assert_success
    assert_output --partial "downgrade detected"
    assert_upgrade "16"
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.0"
    [ ! -e "$PGDATA/should-be-removed" ]
    [ ! -e "$PGDATA/.stats-pending" ]
    [ ! -e "$TEST_DIR/upgrade-ran" ]
    [ ! -e "$TEST_DIR/refresh-ran" ]
}

@test "main - statistics shutdown failure after upgrade preserves ACS version" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16
    echo "5.1" > "$PGDATA/.scanner-db-version"

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    mock_executable "$NEW_BINARIES/pg_ctl" '[[ "$*" != *stop* ]]'
    mock_executable "$NEW_BINARIES/vacuumdb" 'exit 1'
    export BUILD_TAG="5.2.0"

    run bash -c 'source "$1"; _main' _ "${BATS_TEST_DIRNAME}/../scripts/init-entrypoint.sh"
    assert_failure
    assert_output --partial "Could not stop the refresh server"
    assert_upgrade "16"
    assert_equal "$(cat "$PGDATA/.scanner-db-version")" "5.1"
    [ -e "$PGDATA/.stats-pending" ]
}

# Optimizer statistics
@test "upgrade - marks statistics pending" {
    init_pgdata
    given_data_version "$PGDATA" 15
    setup_binaries 15 16

    run maybe-upgrade "$TEST_DIR/pgsql"
    assert_success
    assert_upgrade "16"
    [ -e "$PGDATA/.stats-pending" ]
}

@test "statistics - refreshes and clears the marker" {
    given_pending_statistics

    run maybe-refresh-statistics
    assert_success
    [ -e "$TEST_DIR/analyzed" ]
    [ ! -e "$PGDATA/.stats-pending" ]
}

@test "statistics - does nothing without the marker" {
    given_pending_statistics
    rm -f "$PGDATA/.stats-pending"

    run maybe-refresh-statistics
    assert_success
    [ ! -e "$TEST_DIR/started" ]
    [ ! -e "$TEST_DIR/analyzed" ]
}

@test "statistics - keeps the marker when vacuumdb fails" {
    given_pending_statistics
    mock_executable "$NEW_BINARIES/vacuumdb" 'exit 1'

    run maybe-refresh-statistics
    assert_success
    [ -e "$PGDATA/.stats-pending" ]
}

@test "statistics - keeps the marker when the cluster will not start" {
    given_pending_statistics
    mock_executable "$NEW_BINARIES/pg_ctl" 'exit 1'

    run maybe-refresh-statistics
    assert_success
    [ ! -e "$TEST_DIR/analyzed" ]
    [ -e "$PGDATA/.stats-pending" ]
}

# Clean shutdown
@test "clean-shutdown succeeds when cluster shuts down" {
    mkdir -p "$PGDATA"
    local old_bin="$TEST_DIR/pgsql/postgresql-15/bin"
    mock_executable "$old_bin/pg_ctl" ''
    mock_executable "$old_bin/pg_isready" ''
    mock_executable "$old_bin/pg_controldata" "echo 'Database cluster state:               shutdown'"

    PATH="$old_bin:$PATH" run clean-shutdown
    assert_success
}

@test "clean-shutdown fails when cluster state != shutdown" {
    mkdir -p "$PGDATA"
    local old_bin="$TEST_DIR/pgsql/postgresql-15/bin"
    mock_executable "$old_bin/pg_ctl" ''
    mock_executable "$old_bin/pg_isready" 'exit 1'

    PATH="$old_bin:$PATH" run clean-shutdown
    [ "$status" -ne 0 ]
}

# Recovery from interrupts
@test "recover-interrupted-upgrade - restore pre-upgrade dir" {
    mkdir -p "$PGDATA-old"
    touch "$PGDATA-old/.init-ready"
    echo "data" > "$PGDATA-old/file.txt"

    recover-interrupted-upgrade
    [ -f "$PGDATA/file.txt" ]
    [ ! -d "$PGDATA-old" ]
}

@test "recover-interrupted-upgrade - clean up post-upgrade" {
    mkdir -p "$PGDATA-old" "$PGDATA"
    touch "$PGDATA/.init-ready"

    recover-interrupted-upgrade
    [ ! -d "$PGDATA-old" ]
}

@test "recover-interrupted-upgrade - discard orphaned PGDATA-new" {
    mkdir -p "$PGDATA-new"
    echo "stale" > "$PGDATA-new/marker"

    recover-interrupted-upgrade
    [ ! -d "$PGDATA-new" ]
}
