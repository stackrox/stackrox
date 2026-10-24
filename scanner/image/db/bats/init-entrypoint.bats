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

assert_upgrade() {
    local expected_version=$1
    [ -f "$PGDATA/.init-ready" ] || return 1
    [[ "$(cat "$PGDATA/PG_VERSION")" == "$expected_version" ]] || return 1
    [ ! -d "$PGDATA-new" ] || return 1
    [ ! -d "$PGDATA-old" ] || return 1
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
