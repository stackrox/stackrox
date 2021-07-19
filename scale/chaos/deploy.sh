#!/bin/bash

set -euo pipefail

gitroot="$(git rev-parse --show-toplevel)"
[[ -n "${gitroot}" ]] || { echo >&2 "Could not determine git root!"; exit 1; }

export TAG="$(make --no-print-directory --quiet -C "${gitroot}" tag)"

dir="$(dirname "${BASH_SOURCE[0]}")"
envsubst < "${dir}/deploy.yaml" | kubectl apply -f -
