#!/bin/bash
# Post-job cleanup for speedracer gate.
# Called only when job.status==failure. Resolves the stuck in_progress
# check-run so branch protection is not blocked and sibling monitors
# do not treat a failed copy as a winner.
#
# Requires env vars: SPEC_REPO, GH_TOKEN (set by the gate action).

spec_id=$(cat /tmp/speedracer-check-id 2>/dev/null | head -1 | tr -d '[:space:]')
[[ "$spec_id" =~ ^[0-9]+$ ]] || exit 0

cur=$(gh api "repos/${SPEC_REPO}/check-runs/${spec_id}" --jq '.status' 2>/dev/null || echo "")
if [[ "$cur" == "in_progress" ]]; then
  echo "Post-job cleanup: resolving check-run ${spec_id} → completed/failure"
  gh api "repos/${SPEC_REPO}/check-runs/${spec_id}" \
    -X PATCH \
    -f status=completed \
    -f conclusion=failure \
    -f 'output[title]=Post-job speedracer cleanup' \
    -f 'output[summary]=Job failed after gate; check-run resolved to unblock branch protection.' \
    2>/dev/null || true
fi
