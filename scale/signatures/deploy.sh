#!/usr/bin/env bash

set -euo pipefail

# This will deploy a cron-job to K8S which will trigger an update to signature integrations within central each 15 mins.
# If no signature integration exists, nothing will be done.

TEST_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$TEST_ROOT/tests/e2e/lib.sh"

require_environment "ROX_PASSWORD"

# Substitute ROX_PASSWORD within the shell script that will be used to trigger signature integration updates.
# Deploy the CRON job..
dir="$(dirname "${BASH_SOURCE[0]}")"
envsubst '${ROX_PASSWORD}' < "${dir}/deploy.yaml" | kubectl create -f -
