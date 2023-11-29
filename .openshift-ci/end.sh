#!/usr/bin/env bash

# The final script executed for openshift/release CI jobs.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
# shellcheck source=../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"

set -euo pipefail

info "And it shall end"

if [[ -f "${SHARED_DIR:-}/shared_env" ]]; then
    # shellcheck disable=SC1091
    source "${SHARED_DIR:-}/shared_env"
fi

openshift_ci_mods
openshift_ci_import_creds

# There will be no outcome if dispatch.sh never ran e.g. pod runner could not be
# scheduled, or the job was canceled before it ran e.g. short commits on a PR,
# or the cluster create failed, or...
set_job_record_outcome_if_missing "${OUTCOME_FAILED}"

update_job_record stopped_at "CURRENT_TIMESTAMP()"
