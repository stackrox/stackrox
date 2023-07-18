#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

set -e

die() {
	echo >&2 "$@"
	exit 1
}

echo "Building with platform linux/${GOARCH}"

if command -v docker-buildx; then
    docker-buildx build --platform "linux/${GOARCH}" --load $@
fi

docker buildx build --platform "linux/${GOARCH}" --load $@
