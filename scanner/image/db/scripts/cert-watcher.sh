#!/usr/bin/env bash
set -uo pipefail

CERT_DIR="${ROX_CERT_DIR:-/run/secrets/stackrox.io/certs}"
PGDATA="${PGDATA:-/var/lib/postgresql/data/pgdata}"
POLL_INTERVAL="${ROX_CERT_POLL_INTERVAL:-5s}"

# Timestamp format matches PostgreSQL.
log() { echo "$(date -u '+%Y-%m-%d %H:%M:%S.%3N UTC') cert-watcher: $*"; }

if ! command -v pg_ctl &>/dev/null; then
    log "ERROR: pg_ctl not found, certificate reload will not work"
    sleep infinity
fi

log "watching ${CERT_DIR} for changes (interval: ${POLL_INTERVAL}, pgdata: ${PGDATA})"

HASH=""
while true; do
    NEW_HASH=$(cat "$CERT_DIR"/server.crt "$CERT_DIR"/server.key 2>/dev/null | md5sum | awk '{print $1}')
    if [ -n "$NEW_HASH" ] && [ "$NEW_HASH" != "$HASH" ]; then
        if [ -n "$HASH" ]; then
            log "TLS certificates changed, reloading PostgreSQL"
            if ! pg_ctl reload -D "$PGDATA"; then
                log "pg_ctl reload failed, will retry"
                continue
            fi
        fi
        HASH="$NEW_HASH"
    fi
    sleep "$POLL_INTERVAL"
done
