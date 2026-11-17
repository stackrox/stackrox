#!/usr/bin/env bash

# init-entrypoint.sh initializes the DB if it does not exist, and upgrades an
# existing cluster in place when this image ships PostgreSQL binaries newer
# than the data directory on disk.
#
# A failed upgrade leaves $PGDATA untouched and returns 0; the server starts
# with matching binaries and the upgrade is retried on restart. A failed file
# system operation returns non-zero instead, because a volume that cannot be
# written to cannot host a running server either.
#
# On version downgrade (major.minor decrease), it wipes and reinitializes the
# database so that the older scanner binary does not encounter incompatible
# forward-migrated schema.

set -Eeo pipefail

# File flag that marks $PGDATA as initialized.
INIT_READY_FILE=".init-ready"

# File flag that marks a statistics refresh is needed.
STATS_PENDING_FILE=".stats-pending"

# File that records the ACS major.minor version of the database.
ACS_VERSION_FILE=".scanner-db-version"

# extract-major-minor strips a version string down to its major.minor
# components. Accepts formats like "4.8.1", "5.1.0-rc.2", or
# "5.2.x-23-gabcdef".  Returns "" for unparseable input.
extract-major-minor() {
    local ver=${1:-}
    if [[ "$ver" =~ ^([0-9]+\.[0-9]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    fi
}

# is-downgrade returns 0 (true) when $current is strictly older than $stored,
# comparing only major.minor.  Returns 1 when versions are equal, current is
# newer, or either value is empty / unparseable.
is-downgrade() {
    local stored=${1:-}
    local current=${2:-}

    [ -z "$stored" ] || [ -z "$current" ] && return 1
    [ "$stored" = "$current" ] && return 1

    local oldest
    oldest=$(printf '%s\n%s\n' "$stored" "$current" | sort -V | head -1)
    [ "$oldest" = "$current" ] && return 0
    return 1
}

# reset-on-downgrade invalidates initialization when the stored version is unsafe.
reset-on-downgrade() {
    local current_mm=$1
    local init_check="$PGDATA/$INIT_READY_FILE"
    local version_file="$PGDATA/$ACS_VERSION_FILE"

    # Skip version checks when the current version or stored state is unknown.
    [ -n "$current_mm" ] || return 0
    [ -e "$init_check" ] || return 0
    [ -f "$version_file" ] || return 0

    local stored_mm
    stored_mm=$(cat "$version_file")
    if [[ ! "$stored_mm" =~ ^[0-9]+\.[0-9]+$ ]]; then
        echo >&2 "WARNING: malformed version in $version_file: '$stored_mm'. Reinitializing database."
        rm -f "$init_check"
        return
    fi

    is-downgrade "$stored_mm" "$current_mm" || return 0
    echo "Scanner V4 DB downgrade detected ($stored_mm -> $current_mm). Reinitializing database."
    rm -f "$init_check"
}

# set-acs-version atomically updates the ACS version marker when its value
# changes, leaving the existing marker intact if the write is interrupted.
set-acs-version() {
    local datadir=${1:?missing first required argument: datadir}
    local current_acs_version=${2:-}
    local version_file="$datadir/$ACS_VERSION_FILE"

    # Remove temporary markers left by an interrupted earlier write.
    rm -f "$datadir/$ACS_VERSION_FILE".?????? 2>/dev/null || true

    [ -n "$current_acs_version" ] || return 0
    if [ -f "$version_file" ] && [ "$(cat "$version_file")" = "$current_acs_version" ]; then
        return 0
    fi

    local tmp_version_file
    tmp_version_file=$(mktemp "$datadir/$ACS_VERSION_FILE.XXXXXX") ||
        { echo "Could not create a temporary ACS version marker in $datadir, aborting."; return 1; }
    if ! printf '%s\n' "$current_acs_version" > "$tmp_version_file"; then
        rm -f "$tmp_version_file"
        echo "Could not write the ACS version marker in $datadir, aborting."
        return 1
    fi
    mv "$tmp_version_file" "$version_file" ||
        { rm -f "$tmp_version_file"; echo "Could not update the ACS version marker in $datadir, aborting."; return 1; }
}

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

# maybe-refresh-statistics runs the analyze an upgrade left pending.
maybe-refresh-statistics() {
    local pending="$PGDATA/$STATS_PENDING_FILE"

    [ -e "$pending" ] || return 0

    echo "Refreshing optimizer statistics."
    if ! pg_ctl -D "$PGDATA" start -w --timeout 86400 -o "-h 127.0.0.1"; then
        echo "Could not start the upgraded cluster, retrying on the next start."
        return 0
    fi
    if PGPASSWORD="$(cat "$POSTGRES_PASSWORD_FILE")" \
            vacuumdb --host 127.0.0.1 --all --analyze-only --jobs 4; then
        rm -f "$pending" || true
        echo "Statistics refreshed, shutting down the cluster."
    else
        echo "vacuumdb failed, retrying on the next start."
    fi
    if ! pg_ctl -D "$PGDATA" stop -w --timeout 600; then
        echo "Could not stop the refresh server, retrying on the next start."
        return 1
    fi
}

# recover-interrupted-upgrade cleans up upgrade artifacts left behind by pod
# crashes. It discards the new directory, and restores the pre-upgrade one if
# the crash happened mid-swap. It returns non-zero when a directory can be
# neither removed nor restored.
recover-interrupted-upgrade() {
    local new_datadir="${PGDATA}-new"
    local old_datadir="${PGDATA}-old"

    # Discard orphaned new directory
    rm -rf "$new_datadir" || { echo "Could not remove $new_datadir."; return 1; }

    [ -d "$old_datadir" ] || return 0

    if [ -e "$PGDATA/$INIT_READY_FILE" ]; then
        echo "Removing the pre-upgrade data directory $old_datadir."
        rm -rf "$old_datadir" || { echo "Could not remove $old_datadir."; return 1; }
        return 0
    fi

    # $PGDATA is missing, so restore the old directory, but only the one this
    # script moved aside.
    if [ -e "$old_datadir/$INIT_READY_FILE" ]; then
        echo "An upgrade was interrupted mid-swap, restoring $old_datadir."
        rm -rf "$PGDATA" && mv "$old_datadir" "$PGDATA" && return 0
    fi

    echo "Could not restore from $old_datadir, refusing to start without $PGDATA."
    return 1
}

# maybe-upgrade upgrades $PGDATA in place when this image's binaries are newer
# than the cluster on disk.
maybe-upgrade() {
    # The base image ships the previous major under old_binaries_path.
    local old_binaries_path=${1:-/usr/lib64/pgsql}
    local current_acs_version=${2:-}
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
    if ! chmod 0700 "$PGDATA"; then
        echo "Could not set permissions on $PGDATA, staying on PostgreSQL $data_version."
        return 0
    fi

    echo "Making sure PostgreSQL is shut down cleanly."
    if ! PATH="$OLD_BINDIR:$PATH" clean-shutdown; then
        echo "Cluster was not shut down cleanly, cannot upgrade."
        return 0
    fi

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

    # pg_upgrade does not transfer our ACS version marker. Prefer the current
    # image version, falling back to the old marker if it is unavailable. Write
    # it in the new directory before marking it ready, so an interrupted write
    # cannot corrupt the marker in the existing cluster.
    if [ -n "$current_acs_version" ]; then
        set-acs-version "$NEW_DATADIR" "$current_acs_version" || return 1
    elif [ -f "$PGDATA/$ACS_VERSION_FILE" ]; then
        cp "$PGDATA/$ACS_VERSION_FILE" "$NEW_DATADIR/$ACS_VERSION_FILE" ||
            echo "Could not preserve the ACS version in $NEW_DATADIR; continuing without the marker."
    fi

    # The sentinel is written before the swap so that an interruption leaves a
    # state recover-interrupted-upgrade can undo.
    touch "$NEW_DATADIR/$INIT_READY_FILE" ||
        { echo "Could not mark $NEW_DATADIR ready, aborting."; return 1; }
    # Mark statistics pending; the refresh clears it on success.
    touch "$NEW_DATADIR/$STATS_PENDING_FILE" ||
        { echo "Could not mark statistics pending, aborting."; return 1; }
    mv "$PGDATA" "$old_datadir" ||
        { echo "Could not move $PGDATA aside, aborting."; return 1; }
    mv "$NEW_DATADIR" "$PGDATA" ||
        { echo "Could not move $NEW_DATADIR into place, restoring $old_datadir on the next start."; return 1; }
    rm -rf "$old_datadir" || echo "Could not remove $old_datadir, removing it on the next start."

    echo "Upgraded to PostgreSQL $binary_version."
}

_is_sourced() {
    [[ "${BASH_SOURCE[0]}" != "${0}" ]]
}

_main() {
    local init_check="$PGDATA/$INIT_READY_FILE"
    local init_bundle="/db-init.dump.zst"
    local current_acs_version
    current_acs_version=$(extract-major-minor "${BUILD_TAG:-}")

    recover-interrupted-upgrade
    reset-on-downgrade "$current_acs_version"

    if [ -e "$init_check" ]; then
        maybe-upgrade /usr/lib64/pgsql "$current_acs_version"
        maybe-refresh-statistics
        set-acs-version "$PGDATA" "$current_acs_version"
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

    set-acs-version "$PGDATA" "$current_acs_version"
    touch "$init_check"
}

if ! _is_sourced; then
    _main "$@"
fi
