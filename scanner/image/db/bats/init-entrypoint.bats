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
    mock_executable "$NEW_BINARIES/initdb" "while [[ \$# -gt 0 ]]; do
    if [[ \"\$1\" == --pgdata ]]; then
        datadir=\"\$2\"
        shift 2
    else
        shift
    fi
done
mkdir -p \"\$datadir\"
echo \"$new_version\" > \"\$datadir/PG_VERSION\""

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

# Fresh initialization
@test "fresh init creates database" {
    mkdir -p "$PGDATA"
    init-cluster "$PGDATA"
    [ -f "$PGDATA/PG_VERSION" ]
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

@test "statistics - a pending refresh is retried on a later start" {
    given_pending_statistics

    # The whole point of a separate marker: by now maybe-upgrade sees matching
    # versions and returns early, so _main has to drive the refresh.
    run _main
    assert_success
    [ -e "$TEST_DIR/analyzed" ]
    [ ! -e "$PGDATA/.stats-pending" ]
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
