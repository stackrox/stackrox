#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$ROOT/scripts/lib.sh"

# TODO opts - -flavor to distinguish between qa-tests-backend, non-groovy, ui
#             -suite? -case?
#             -pre-deployed?

# TODO safety check that this is running against an infra gke-default cluster -
# ignore if 'pre-deployed'

# TODO teardown - probably optional but defaults to true for 'full' test runs

# TODO got image? - for a 'full' test run we might want to poll for 'canonical'
# images - poll_for_system_test_images - warn about `make tag` ~ /-dirty/?

# TODO has vault?

# TODO log in to vault? - can we CLI access to the collections? helpme.

# TODO can access e2e creds? more helpme.

# TODO import all the e2e secrets from vault. be secure.

# TODO space for required env settings e.g. CI=true

# --flavor qa
"$ROOT/qa-tests-backend/scripts/run-part-1.sh"
# --flavor e2e
#"$ROOT/tests/e2e/run.sh"
