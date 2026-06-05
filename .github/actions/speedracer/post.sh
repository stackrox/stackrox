#!/bin/bash
# Post-job handler for speedracer gate.
# Registered only when the gate step completes (i.e. only on passing/winning copies).
# Cancelled copies (slow runner or monitor-killed) never reach the gate's step 2,
# so this post step is never registered for them — no guard needed.
#
# success → patch check-run to completed/success (declares the winner)
# failure → patch check-run to completed/failure (unblocks branch protection)
# cancelled → unreachable (see above)
#
# Env: JOB_STATUS, GH_TOKEN, SPEC_REPO, SPEC_SHA (injected by post: line)

check_id=$(cat /tmp/speedracer-check-id | head -1 | tr -d '[:space:]')
check_name=$(cat /tmp/speedracer-check-name | head -1 | tr -d '\n')
my_copy=$(cat /tmp/speedracer-copy | head -1 | tr -d '\n')

if [[ "$JOB_STATUS" == "success" ]]; then
  echo "Speedracer post: copy '${my_copy}' won — marking check-run complete."
  if [[ "$check_id" =~ ^[0-9]+$ ]]; then
    gh api "repos/${SPEC_REPO}/check-runs/${check_id}" \
      -X PATCH \
      -f status=completed \
      -f conclusion=success \
      -f "output[title]=Speedracer complete" \
      -f "output[summary]=Copy ${my_copy} won." \
      || echo "::warning::Failed to update check-run (non-fatal)"
  else
    gh api "repos/${SPEC_REPO}/check-runs" \
      -f name="${check_name}" \
      -f head_sha="${SPEC_SHA}" \
      -f status=completed \
      -f conclusion=success \
      -f "output[title]=Speedracer complete" \
      -f "output[summary]=Copy ${my_copy} won." \
      || echo "::warning::Failed to post check-run (non-fatal)"
  fi
elif [[ "$JOB_STATUS" == "failure" ]]; then
  echo "Speedracer post: job failed — resolving check-run to unblock branch protection."
  [[ "$check_id" =~ ^[0-9]+$ ]] || exit 0
  cur=$(gh api "repos/${SPEC_REPO}/check-runs/${check_id}" --jq '.status' || echo "")
  if [[ "$cur" == "in_progress" ]]; then
    gh api "repos/${SPEC_REPO}/check-runs/${check_id}" \
      -X PATCH \
      -f status=completed \
      -f conclusion=failure \
      -f 'output[title]=Speedracer post-job cleanup' \
      -f 'output[summary]=Job failed; check-run resolved to unblock branch protection.' \
      || true
  fi
fi
