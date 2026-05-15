#!/usr/bin/env bash
set -uo pipefail

CERT_DIR="${ROX_CERT_DIR:-/run/secrets/stackrox.io/certs}"
PGDATA="${PGDATA:-/var/lib/postgresql/data/pgdata}"
POLL_INTERVAL="${ROX_CERT_POLL_INTERVAL:-5s}"

if ! command -v pg_ctl &>/dev/null; then
    echo "cert-watcher: ERROR: pg_ctl not found, certificate reload will not work"
    sleep infinity
fi

echo "cert-watcher: watching ${CERT_DIR} for changes (interval: ${POLL_INTERVAL})"

HASH=""
while true; do
    sleep "$POLL_INTERVAL"
    NEW_HASH=$(md5sum "$CERT_DIR"/server.crt "$CERT_DIR"/server.key 2>/dev/null | md5sum | awk '{print $1}')
    if [ -n "$NEW_HASH" ] && [ "$NEW_HASH" != "$HASH" ]; then
        if [ -n "$HASH" ]; then
            echo "cert-watcher: TLS certificates changed, reloading PostgreSQL"
            if ! pg_ctl reload -D "$PGDATA"; then
                echo "cert-watcher: pg_ctl reload failed"
            fi
        fi
        HASH="$NEW_HASH"
    fi
done
