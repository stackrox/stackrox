#!/usr/bin/env bash

set -ex

echo "Running with platform linux/${GOARCH}"
docker run --platform "linux/${GOARCH}" -e GOARCH="${GOARCH}" -e PLATFORM="linux/${GOARCH}" "$@"
