#!/usr/bin/env bash

# init-entrypoint.sh initializes the DB if it does not exist, and upgrades an
# existing cluster in place when this image ships PostgreSQL binaries newer
# than the data directory on disk.
#
# Upgrade failures do not fail the container. An upgrade Failure leaves
# $PGDATA untouched and returns 0; the server starts with matching binaries
# and the upgrade is retried on restart.

set -Eeo pipefail

# File flag that marks $PGDATA as initialized.
INIT_READY_FILE=".init-ready"

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

# init-cluster creates an empty cluster in the given data directory.
init-cluster() {
    local datadir=${1:?missing first required argument: datadir}

    initdb --pgdata "$datadir" \
           --auth-host="$POSTGRES_HOST_AUTH_METHOD" \
           --auth-local="$POSTGRES_HOST_AUTH_METHOD" \
           --pwfile "$POSTGRES_PASSWORD_FILE" \
           --data-checksums 2>&1
}

# pg-major-version echoes the major version of the PostgreSQL binaries on PATH.
pg-major-version() {
    postgres -V | sed 's/postgres (PostgreSQL) \([0-9]*\).*/\1/'
}

# has-available-space returns 0 if the file system holding path has enough
# free space to copy path's contents.
has-available-space() {
    local path=${1:?missing first required argument: path}
    local required available

    required=$(du -sk "$path" | awk '{print $1}')
    available=$(df -Pk "$path" | tail -n 1 | awk '{print $4}')
    if [ -z "$available" ]; then
        echo "Cannot determine free space at $path."
        return 1
    fi

    echo "Free space at $path: ${available}K, required: ${required}K."
    [ "$available" -ge "$required" ]
}

# clean-shutdown starts and stops the cluster,
# leaving it in the cleanly shut down state pg_upgrade requires.
clean-shutdown() {
    local state

    pg_ctl start -w --timeout 86400 -o "-h 127.0.0.1" 2>&1 || return 1
    pg_isready -h 127.0.0.1 2>&1 || return 1
    pg_ctl stop -w 2>&1 || return 1

    state=$(pg_controldata -D "$PGDATA" 2>&1 |
                grep "Database cluster state" |
                awk -F ':' '{print $2}' |
                tr -d '[:space:]')

    if [ "$state" != "shutdown" ]; then
        echo "Cluster state is \"$state\", expected \"shutdown\"."
        return 1
    fi
}

# pg-upgrade runs pg_upgrade with the transfer mode and optionally --check.
pg-upgrade() {
    PGPASSWORD="$(cat "$POSTGRES_PASSWORD_FILE")" \
        "${NEW_BINDIR}/pg_upgrade" \
            --old-bindir="$OLD_BINDIR" \
            --new-bindir="$NEW_BINDIR" \
            --old-datadir="$PGDATA" \
            --new-datadir="$NEW_DATADIR" \
            -j 4 \
            "$@"
}

# recover-interrupted-upgrade cleans up upgrade artifacts left behind by pod
# crashes. It restores the pre-upgrade data directory if the crash happened
# mid-swap, or discards the new directory if it happened before the swap.
recover-interrupted-upgrade() {
    local new_datadir="${PGDATA}-new"
    local old_datadir="${PGDATA}-old"

    # Discard orphaned new directory
    [ -d "$new_datadir" ] && rm -rf "$new_datadir"

    # Restore old directory
    [ -d "$old_datadir" ] || return 0

    if [ -e "$PGDATA/$INIT_READY_FILE" ]; then
        echo "Removing the pre-upgrade data directory $old_datadir."
        rm -rf "$old_datadir"
        return 0
    fi

    echo "An upgrade was interrupted mid-swap, restoring $old_datadir."
    rm -rf "$PGDATA"
    mv "$old_datadir" "$PGDATA"
}

# maybe-upgrade upgrades $PGDATA in place when this image's binaries are newer
# than the cluster on disk.
maybe-upgrade() {
    # The base image ships the previous major under old_binaries_path.
    local old_binaries_path=${1:-/usr/lib64/pgsql}
    local old_datadir="${PGDATA}-old"
    local major_regex='^[0-9]+$'
    local binary_version data_version mode

    export NEW_DATADIR="${PGDATA}-new"

    if [ ! -s "$PGDATA/PG_VERSION" ]; then
        echo "Warning: PG_VERSION is missing or empty in $PGDATA, skipping upgrade."
        return 0
    fi

    data_version=$(< "$PGDATA/PG_VERSION") || true
    if ! [[ "$data_version" =~ $major_regex ]]; then
        echo "Cannot read a valid major version from $PGDATA/PG_VERSION: \"$data_version\"."
        return 0
    fi

    binary_version=$(pg-major-version) || true
    if ! [[ "$binary_version" =~ $major_regex ]]; then
        echo "Cannot read a valid major version from postgres -V: \"$binary_version\"."
        return 0
    fi
    export NEW_BINDIR="${NEW_BINARIES:-/usr/bin}"

    if [ "$data_version" -ge "$binary_version" ]; then
        echo "Data is PostgreSQL $data_version, binaries are PostgreSQL $binary_version: no upgrade needed."
        return 0
    fi

    export OLD_BINDIR="${old_binaries_path}/postgresql-${data_version}/bin"
    if [ ! -x "${OLD_BINDIR}/pg_ctl" ]; then
        echo "No PostgreSQL $data_version binaries in $OLD_BINDIR, cannot upgrade."
        return 0
    fi
    echo "Data is PostgreSQL $data_version, binaries are PostgreSQL $binary_version: upgrading..."

    # pg_upgrade rejects group permissions on the data directory.
    chmod 0700 "$PGDATA"

    echo "Making sure PostgreSQL is shut down cleanly."
    if ! PATH="$OLD_BINDIR:$PATH" clean-shutdown; then
        echo "Cluster was not shut down cleanly, cannot upgrade."
        return 0
    fi

    # Guard init-cluster for initdb failures.
    # Log and continue to boot from existing data.
    if ! init-cluster "$NEW_DATADIR"; then
        echo "Could not initialize $NEW_DATADIR, staying on PostgreSQL $data_version."
        rm -rf "$NEW_DATADIR"
        return 0
    fi

    # Probe transfer modes: try --clone first, or --copy if space available.
    # Link mode breaks the old cluster on upgrade failure; recovery requires
    # backup restoration. With clone or copy mode, we keep old data safe, so the
    # server can fallback to the old data.
    mode=""
    if pg-upgrade --clone --check 2>&1; then
        mode="--clone"
    else
        if ! has-available-space "$PGDATA"; then
            echo "Not enough free space to copy the cluster."
        elif ! pg-upgrade --copy --check 2>&1; then
            echo "pg_upgrade --copy --check failed; see logs above."
        else
            mode="--copy"
        fi
    fi

    if [ -z "$mode" ]; then
        echo "No usable transfer mode, staying on PostgreSQL $data_version."
        rm -rf "$NEW_DATADIR"
        return 0
    fi

    echo "Upgrading with $mode."
    if ! pg-upgrade "$mode" 2>&1; then
        echo "Upgrade failed, staying on PostgreSQL $data_version."
        find "$NEW_DATADIR" -name pg_upgrade_server.log -exec cat {} \; || true
        rm -rf "$NEW_DATADIR"
        return 0
    fi

    # The server reads config_file and hba_file from outside the data
    # directory, so $PGDATA/*.conf is not carried over.
    #
    # The sentinel is written before the swap so that an interruption leaves a
    # state recover-interrupted-upgrade can undo.
    touch "$NEW_DATADIR/$INIT_READY_FILE"
    mv "$PGDATA" "$old_datadir"
    mv "$NEW_DATADIR" "$PGDATA"
    rm -rf "$old_datadir"

    echo "Upgraded to PostgreSQL $binary_version."
}

_is_sourced() {
    [[ "${BASH_SOURCE[0]}" != "${0}" ]]
}

_main() {
    local init_check="$PGDATA/$INIT_READY_FILE"
    local init_bundle="/db-init.dump.zst"

    recover-interrupted-upgrade

    if [ -e "$init_check" ]; then
        maybe-upgrade /usr/lib64/pgsql
        return
    fi

    # Ensure DB is clean to avoid blocking on failed initialization attempts.
    rm -rf "$PGDATA"

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

if ! _is_sourced; then
    _main "$@"
fi
