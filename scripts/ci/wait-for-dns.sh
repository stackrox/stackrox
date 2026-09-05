#!/usr/bin/env bash
# Wait until the given hosts resolve, to ride out transient DNS flakes in CI
# runner pods ("Name or service not known" from resolver blips).
# Usage: wait-for-dns.sh <timeout_seconds> <host> [<host>...]

set -euo pipefail

if [[ "$#" -lt 2 ]]; then
    echo "usage: $0 <timeout_seconds> <host> [<host>...]" >&2
    exit 1
fi

timeout="$1"
shift
deadline=$(( $(date +%s) + timeout ))

for host in "$@"; do
    until getent hosts "$host" >/dev/null 2>&1; do
        if [[ "$(date +%s)" -ge "$deadline" ]]; then
            echo "ERROR: DNS still not resolving '${host}' after ${timeout}s" >&2
            exit 1
        fi
        echo "DNS not resolving '${host}' yet; retrying in 5s" >&2
        sleep 5
    done
done
