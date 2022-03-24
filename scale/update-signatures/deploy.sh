#!/usr/bin/env bash

set -euo pipefail

# This will deploy a cron-job to K8S which will trigger an update to signature integrations within central each 15 mins.
# If no signature integration exists, nothing will be done.

die() {
  echo >&2 "$@"
  exit 1
}

[[ -z "${ROX_PASSWORD}" ]] && die "Required env variable ROX_PASSWORD not set"

# Substitute ROX_PASSWORD within the shell script that will be used to trigger signature integration updates.
# Deploy the CRON job..
dir="$(dirname "${BASH_SOURCE[0]}")"
envsubst '${ROX_PASSWORD}' < "${dir}/deploy.yaml" | kubectl create -f -
