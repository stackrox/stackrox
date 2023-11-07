#!/usr/bin/env bash
set -eou pipefail

die() {
    echo >&2 "$@"
    exit 1
}

log() {
    echo "$*" >&2
}
