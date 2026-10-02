#!/usr/bin/env bash

# init-entrypoint.sh initializes the DB if it does not exist, and upgrades an
# existing cluster in place when this image ships PostgreSQL binaries newer
# than the data directory on disk.
#
# The upgrade is deliberately non-fatal. Whenever it cannot be performed
# safely, $PGDATA is left untouched and this script still exits successfully:
# docker-entrypoint.sh then starts the server with the binaries matching the
# data directory, and the upgrade is retried on the next restart. Discarding
# the database instead is not an option, since the matcher reports
# FailedPrecondition while it reloads the vulnerability store, which fails
# scans outright rather than merely slowing them down.
#
# Every soft-failure path must "return 0", because maybe-upgrade is called
# unconditionally from main, which is the script's last command -- a non-zero
# return would exit the init container non-zero and crashloop the pod, which
# is precisely what this exists to prevent. Helpers that can fail
# (clean-shutdown, pg-upgrade) are always called in a conditional context,
# where bash disables "set -e" inside the function body; they therefore use
# explicit "|| return 1" rather than relying on it.

set -Eeo pipefail

# Binaries of the PostgreSQL version this image is built on. Binaries of the
# previous major version live in /usr/lib64/pgsql/postgresql-<version>/bin
# (see POSTGRESQL_PREV_VERSION in the base image).
NEW_BINARIES="/usr/bin"

# load-init-bundle calls pg_restore on the database init bundle.
load-init-bundle() {
    local db=${1:?missing first required argument: db}
    local bundle=${2:?missing second required argument: f}
    local dump

    dump=$(mktemp)

    zstd >"$dump" -dc "$bundle"
    pg_ctl start
    pg_restore \
        --verbose \
        --format=custom \
        --jobs "$(nproc)" \
        --exit-on-error \
        --dbname "$db" \
        --no-owner \
        "$dump"
    pg_ctl stop
    rm -f "$dump"
}

# init-cluster creates an empty cluster in the given data directory. Both the
# initial cluster and the target of an upgrade go through this function:
# pg_upgrade compares the two clusters' initdb settings and refuses to run if
# they differ, so they must not drift apart.
init-cluster() {
    local datadir=${1:?missing first required argument: datadir}

    initdb --pgdata "$datadir" \
           --auth-host="$POSTGRES_HOST_AUTH_METHOD" \
           --auth-local="$POSTGRES_HOST_AUTH_METHOD" \
           --pwfile "$POSTGRES_PASSWORD_FILE" \
           --data-checksums
}

# pg-major-version echoes the major version of the PostgreSQL binaries on PATH.
pg-major-version() {
    postgres -V | sed 's/postgres (PostgreSQL) \([0-9]*\).*/\1/'
}

# has-available-space succeeds if the file system holding the first argument
# has at least as many free 1K blocks as the second argument asks for.
has-available-space() {
    local path=${1:?missing first required argument: path}
    local required=${2:?missing second required argument: required}
    local available

    available=$(df -Pk "$path" | tail -n 1 | awk '{print $4}')
    if [ -z "$available" ]; then
        echo "Cannot determine the free space at $path."
        return 1
    fi

    echo "Free space at $path: ${available}K, required: ${required}K."
    [ "$available" -ge "$required" ]
}

# clean-shutdown brings the cluster up and down again using the binaries in the
# given directory, so that pg_upgrade finds a cleanly shut down cluster. It
# fails if the cluster cannot be brought into that state.
clean-shutdown() {
    local bindir=${1:?missing first required argument: bindir}
    local state

    "$bindir/pg_ctl" -D "$PGDATA" start -w --timeout 86400 -o "-h 127.0.0.1" || return 1
    "$bindir/pg_isready" -h 127.0.0.1 || return 1
    "$bindir/pg_ctl" -D "$PGDATA" stop -w || return 1

    state=$("$bindir/pg_controldata" -D "$PGDATA" |
                grep "Database cluster state" |
                awk -F ':' '{print $2}' |
                tr -d '[:space:]')

    if [ "$state" != "shutdown" ]; then
        echo "Cluster state is \"$state\", expected \"shutdown\"."
        return 1
    fi
}

# pg-upgrade runs pg_upgrade from the old binaries in $1 into the new data
# directory $2. Remaining arguments carry the transfer mode and optionally
# --check.
#
# pg_upgrade writes delete_old_cluster.sh into the current working directory
# and defaults --socketdir to it as well, so both are pointed somewhere known
# to be writable rather than relying on the image WORKDIR.
pg-upgrade() {
    local old_bindir=${1:?missing first required argument: old-bindir}
    local new_datadir=${2:?missing second required argument: new-datadir}
    shift 2

    (
        cd /tmp || return 1
        PGPASSWORD="$(cat "$POSTGRES_PASSWORD_FILE")" \
            "$NEW_BINARIES/pg_upgrade" \
                --old-bindir="$old_bindir" \
                --new-bindir="$NEW_BINARIES" \
                --old-datadir="$PGDATA" \
                --new-datadir="$new_datadir" \
                --socketdir=/tmp \
                --jobs 4 \
                "$@"
    )
}

# TODO(cdu): review note, remove before the PR. Open question for you: the
# rm -rf and mv below are unguarded, as are the two renames in the swap at
# the end of maybe-upgrade, so a failure there does exit non-zero and
# crashloop. I left them that way deliberately -- if renaming a directory on
# the data volume fails, the volume is broken and coming up on a half-swapped
# state would be worse than not coming up. But it is the same class of thing
# as the init-cluster guard below, so it is a judgement call, not an
# oversight. This is the territory of T9 / ROX-37431.
#
# recover-interrupted-upgrade puts the pre-upgrade data directory back if the
# pod died inside the final swap, in the window between renaming the old
# directory aside and renaming the upgraded one into place. Without this, the
# initialization path in main() would see no data directory and wipe the only
# remaining copy of the database.
recover-interrupted-upgrade() {
    local old_datadir="${PGDATA}-old"

    [ -d "$old_datadir" ] || return 0

    if [ -e "$PGDATA/.init-ready" ]; then
        echo "Removing the pre-upgrade data directory $old_datadir."
        rm -rf "$old_datadir"
        return 0
    fi

    echo "An upgrade was interrupted mid-swap, restoring $old_datadir."
    rm -rf "$PGDATA"
    mv "$old_datadir" "$PGDATA"
}

# maybe-upgrade upgrades $PGDATA in place when this image's binaries are newer
# than the cluster on disk. See the note at the top of this file: every reason
# not to upgrade is logged and returns successfully, leaving $PGDATA as it was.
maybe-upgrade() {
    local new_datadir="${PGDATA}-new"
    local old_datadir="${PGDATA}-old"
    local binary_version data_version old_bindir data_used mode

    if [ ! -s "$PGDATA/PG_VERSION" ]; then
        echo "No PG_VERSION in $PGDATA, skipping the upgrade check."
        return 0
    fi

    binary_version=$(pg-major-version)
    data_version=$(cat "$PGDATA/PG_VERSION")

    if [ "$data_version" -ge "$binary_version" ]; then
        return 0
    fi

    if [ "${FORCE_OLD_BINARIES:-}" = "true" ]; then
        echo "FORCE_OLD_BINARIES is set, staying on PostgreSQL $data_version."
        return 0
    fi

    echo "Data directory is PostgreSQL $data_version, binaries are PostgreSQL $binary_version: upgrading."

    # A leftover upgrade directory means an earlier attempt died before the
    # swap. $PGDATA still holds the complete pre-upgrade cluster, because
    # neither --clone nor --copy modifies it, so the leftover is discardable
    # and the upgrade is simply retried from scratch. central-db exits 1 here
    # and expects an operator to set FORCE_CLEANUP=true; that is a crashloop
    # for what is an unambiguous situation.
    if [ -e "$new_datadir" ]; then
        echo "Discarding $new_datadir, left behind by an interrupted upgrade."
        rm -rf "$new_datadir"
    fi

    old_bindir="/usr/lib64/pgsql/postgresql-${data_version}/bin"
    if [ ! -x "$old_bindir/pg_ctl" ]; then
        echo "No PostgreSQL $data_version binaries in $old_bindir, cannot upgrade."
        return 0
    fi

    # pg_upgrade rejects group permissions on the data directory.
    chmod 0700 "$PGDATA"

    echo "Making sure PostgreSQL is shut down cleanly."
    if ! clean-shutdown "$old_bindir"; then
        echo "Cluster was not shut down cleanly, cannot upgrade."
        return 0
    fi

    # TODO(cdu): review note, remove before the PR. This guard is a deviation
    # from the design doc, which calls init-cluster bare here (section 4.2).
    # Bare is a real bug: maybe-upgrade is called from main as a plain
    # command, and main is the script's last command, so under "set -e" a
    # failing initdb exits the init container non-zero and crashloops the pod
    # -- the one outcome section 3.2 forbids. Verified with a probe script.
    # Covered by the "defers when the new cluster cannot be initialized" bats
    # case in ROX-37427.
    #
    # Guarded like every other fallible step here, and for the same reason:
    # an unguarded failure under "set -e" would exit this script non-zero and
    # crashloop the pod. initdb is the one call in this path that genuinely
    # fails in the field, when the volume has no room for the new cluster --
    # which is exactly the case the upgrade is supposed to defer on.
    if ! init-cluster "$new_datadir"; then
        echo "Could not initialize $new_datadir, staying on PostgreSQL $data_version."
        rm -rf "$new_datadir"
        return 0
    fi

    # Pick a transfer mode. --clone is close to free on file systems with
    # copy-on-write support, and "pg_upgrade --check --clone" probes that for
    # real rather than guessing from the file system type. --copy needs room
    # for a second full copy of the data. --link is never used: it leaves the
    # old cluster unusable if the upgrade fails halfway through, and with no
    # backup in this path the old cluster is the only thing standing between a
    # failed upgrade and a lost vulnerability store.
    mode=""
    if pg-upgrade "$old_bindir" "$new_datadir" --clone --check; then
        mode="--clone"
    else
        data_used=$(du -sk "$PGDATA" | awk '{print $1}')
        if has-available-space "$PGDATA" "$data_used" &&
               pg-upgrade "$old_bindir" "$new_datadir" --copy --check; then
            mode="--copy"
        fi
    fi

    if [ -z "$mode" ]; then
        echo "No usable transfer mode, staying on PostgreSQL $data_version."
        rm -rf "$new_datadir"
        return 0
    fi

    echo "Upgrading with $mode."
    if ! pg-upgrade "$old_bindir" "$new_datadir" "$mode"; then
        echo "Upgrade failed, staying on PostgreSQL $data_version."
        find "$new_datadir" -name pg_upgrade_server.log -exec cat {} \; || true
        rm -rf "$new_datadir"
        return 0
    fi

    # $PGDATA/*.conf is deliberately not carried over, unlike in central-db.
    # The server runs with an external config file and hba_file, so the copies
    # inside the data directory are unused, and one written by PostgreSQL
    # $data_version may contain settings PostgreSQL $binary_version rejects.
    #
    # The sentinel is written before the swap, and the swap itself is two
    # renames, so an interruption can be recognized and undone. See
    # recover-interrupted-upgrade.
    touch "$new_datadir/.init-ready"
    mv "$PGDATA" "$old_datadir"
    mv "$new_datadir" "$PGDATA"
    rm -rf "$old_datadir"

    echo "Upgraded to PostgreSQL $binary_version."
}

# main runs the main script.
main() {
    local init_check="$PGDATA/.init-ready"
    local init_bundle="/db-init.dump.zst"

    recover-interrupted-upgrade

    if [ -e "$init_check" ]; then
        maybe-upgrade
        return
    fi

    # Ensure DB is clean to avoid blocking on failed initialization attempts.
    rm -rf "$PGDATA" "${PGDATA}-new"

    # Create cluster.
    init-cluster "$PGDATA"

    # Load init bundle, if enabled and the bundle exists.
    if [ "${SCANNER_DB_INIT_BUNDLE_ENABLED:-}" = "true" ] &&
           [ -s /db-init.dump.zst ]; then
        PGPASSWORD="$(cat "$POSTGRES_PASSWORD_FILE")" \
            load-init-bundle postgres "$init_bundle"
    fi

    touch "$init_check"
}

[ "${BASH_SOURCE[0]}" = "$0" ] && main "$@"
