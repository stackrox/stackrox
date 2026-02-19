#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
docker build --platform "linux/${GOARCH}" "$@"
