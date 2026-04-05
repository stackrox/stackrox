#!/usr/bin/env bash
# Deterministically select a subset of Go test packages for sharding.
# Usage: shard-packages.sh <shard_index> <total_shards> <packages...>
#
# Distributes packages round-robin by line number modulo total_shards.
# The package list must be the same across all shards for deterministic splitting.

set -euo pipefail

if [[ $# -lt 3 ]]; then
    echo "Usage: $0 <shard_index> <total_shards> <packages...>" >&2
    exit 1
fi

SHARD_INDEX=$1
TOTAL_SHARDS=$2
shift 2

if [[ "$TOTAL_SHARDS" -le 1 ]]; then
    echo "$@"
    exit 0
fi

# Print one package per line, select this shard's subset via round-robin
i=0
for pkg in "$@"; do
    if (( i % TOTAL_SHARDS == SHARD_INDEX )); then
        echo "$pkg"
    fi
    (( i++ )) || true
done
