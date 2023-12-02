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

# Determin a useful overall job outcome based on state shared from prior steps.
# 'undefined' states mean the step did not run or openshift-ci canceled it.
# Note: is states are set after cancelation they do not propagate.

combined="${CREATE_CLUSTER_OUTCOME:-undefined}-${JOB_DISPATCH_OUTCOME:-undefined}-${DESTROY_CLUSTER_OUTCOME:-undefined}"

case "${combined}" in
    undefined-"${OUTCOME_FAILED}"-*)
        # The job was considered a failure before cluster creation was
        # attempted. This can happen with GKE for example where the cluster
        # create starts after image poll.
        outcome="${OUTCOME_FAILED}"
        ;;
    undefined-"${OUTCOME_PASSED}"-*)
        info "ERROR: unexpected state in end.sh: ${combined}"
        outcome="${OUTCOME_FAILED}"
        ;;
    undefined-undefined-*)
        # The job was interrupted before cluster create could complete. or
        # openshift-ci had a meltdown. cluster destroy might still pass, fail or
        # be canceled.
        outcome="${OUTCOME_CANCELED}"
        ;;
    "${OUTCOME_FAILED}"-*-*)
        # Track cluster create failures
        outcome="${OUTCOME_FAILED}"
        ;;
    "${OUTCOME_PASSED}"-undefined-*)
        # The job was interrupted before the test could complete. or somewhat
        # less likely openshift-ci had a meltdown. 
        outcome="${OUTCOME_CANCELED}"
        ;;
    "${OUTCOME_PASSED}-${OUTCOME_FAILED}"-*)
        outcome="${OUTCOME_FAILED}"
        ;;
    "${OUTCOME_PASSED}-${OUTCOME_PASSED}"-undefined)
        # The job was interrupted before cluster destroy could complete, this is
        # not ideal but we can rely on janitor to clean up, for actionableness
        # track as a passing job.
        outcome="${OUTCOME_PASSED}"
        ;;
    "${OUTCOME_PASSED}-${OUTCOME_PASSED}-${OUTCOME_FAILED}")
        # Track cluster destroy failures as overall job failures.
        outcome="${OUTCOME_FAILED}"
        ;;
    "${OUTCOME_PASSED}-${OUTCOME_PASSED}-${OUTCOME_PASSED}")
        # Perfect score!
        outcome="${OUTCOME_PASSED}"
        ;;
    *)
        info "ERROR: unexpected state in end.sh: ${combined}"
        outcome="${OUTCOME_FAILED}"
        ;;
esac

update_job_record outcome "${outcome}" stopped_at "CURRENT_TIMESTAMP()"
