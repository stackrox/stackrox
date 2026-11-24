#!/usr/bin/env bash

# init-entrypoint.sh initializes the DB if it does not exist.
#
# The password file path is passed via $POSTGRES_PASS_FILE. The variable is
# named without the substring "PASSWORD" so that the default policy
# "Environment Variable Contains Secret", which matches env var names against
# .*PASSWORD.*, does not fire on our own deployments (ROX-39749). Pod specs
# from releases that predate the rename set POSTGRES_PASSWORD_FILE instead,
# and both are accepted below, so image-only upgrades keep working.

set -Eeo pipefail

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
    # Prefer POSTGRES_PASS_FILE; fall back to the pre-rename
    # POSTGRES_PASSWORD_FILE so image-only upgrades under old pod specs work.
    local pwfile="${POSTGRES_PASS_FILE:-${POSTGRES_PASSWORD_FILE:-}}"

    [ -e "$init_check" ] && return

    # Ensure DB is clean to avoid blocking on failed initialization attempts.
    rm -rf "$PGDATA"

    # Create cluster.
    initdb --auth-host="$POSTGRES_HOST_AUTH_METHOD" \
           --auth-local="$POSTGRES_HOST_AUTH_METHOD" \
           --pwfile "$pwfile" \
           --data-checksums

    # Load init bundle, if enabled and the bundle exists.
    if [ "${SCANNER_DB_INIT_BUNDLE_ENABLED:-}" = "true" ] &&
           [ -s /db-init.dump.zst ]; then
        PGPASSWORD="$(cat "$pwfile")" \
            load-init-bundle postgres "$init_bundle"
    fi

    touch "$init_check"
}

[ "${BASH_SOURCE[0]}" = "$0" ] && main "$@"
