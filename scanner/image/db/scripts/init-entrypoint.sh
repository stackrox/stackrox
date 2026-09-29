#!/usr/bin/env bash

# init-entrypoint.sh initializes the DB if it does not exist.
# On version downgrade (major.minor decrease), it wipes and reinitializes the
# database so that the older scanner binary does not encounter incompatible
# forward-migrated schema.

set -Eeo pipefail

# extract_major_minor strips a version string down to its major.minor
# components. Accepts formats like "4.8.1", "5.1.0-rc.2", or
# "5.2.x-23-gabcdef".  Returns "" for unparseable input.
extract_major_minor() {
    local ver=${1:-}
    if [[ "$ver" =~ ^([0-9]+\.[0-9]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    fi
}

# is_downgrade returns 0 (true) when $current is strictly older than $stored,
# comparing only major.minor.  Returns 1 when versions are equal, current is
# newer, or either value is empty / unparseable.
is_downgrade() {
    local stored=${1:-}
    local current=${2:-}

    [ -z "$stored" ] || [ -z "$current" ] && return 1
    [ "$stored" = "$current" ] && return 1

    local oldest
    oldest=$(printf '%s\n%s\n' "$stored" "$current" | sort -V | head -1)
    [ "$oldest" = "$current" ] && return 0
    return 1
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

# main runs the main script.
main() {
    local init_check="$PGDATA/.init-ready"
    local init_bundle="/db-init.dump.zst"
    local version_file="$PGDATA/.scanner-db-version"

    # SCANNER_V4_DB_VERSION is baked into the image at build time (see the DB
    # Dockerfiles), so it reflects the actual running image regardless of how the
    # image is referenced (tag, digest, or override).
    local current_mm
    current_mm=$(extract_major_minor "${SCANNER_V4_DB_VERSION:-}")

    # Detect downgrade: if the stored major.minor is newer than the current
    # one, wipe the database so the older scanner can start clean.
    # Skip version checks entirely when current version is unknown.
    if [ -n "$current_mm" ] && [ -e "$init_check" ] && [ -f "$version_file" ]; then
        local stored_mm
        stored_mm=$(cat "$version_file")
        if [[ ! "$stored_mm" =~ ^[0-9]+\.[0-9]+$ ]]; then
            echo >&2 "WARNING: malformed version in $version_file: '$stored_mm'. Reinitializing database."
            rm -f "$init_check"
        elif is_downgrade "$stored_mm" "$current_mm"; then
            echo "Scanner V4 DB downgrade detected ($stored_mm -> $current_mm). Reinitializing database."
            rm -f "$init_check"
        fi
    fi

    # DB already initialized and no downgrade detected — record version and exit.
    if [ -e "$init_check" ]; then
        if [ -n "$current_mm" ]; then
            echo "$current_mm" > "$version_file"
        fi
        return
    fi

    # Ensure DB is clean to avoid blocking on failed initialization attempts.
    rm -rf "$PGDATA"

    # Create cluster.
    initdb --auth-host="$POSTGRES_HOST_AUTH_METHOD" \
           --auth-local="$POSTGRES_HOST_AUTH_METHOD" \
           --pwfile "$POSTGRES_PASSWORD_FILE" \
           --data-checksums

    # Load init bundle, if enabled and the bundle exists.
    if [ "${SCANNER_DB_INIT_BUNDLE_ENABLED:-}" = "true" ] &&
           [ -s /db-init.dump.zst ]; then
        PGPASSWORD="$(cat "$POSTGRES_PASSWORD_FILE")" \
            load-init-bundle postgres "$init_bundle"
    fi

    if [ -n "$current_mm" ]; then
        echo "$current_mm" > "$version_file"
    fi
    touch "$init_check"
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
    main "$@"
fi
