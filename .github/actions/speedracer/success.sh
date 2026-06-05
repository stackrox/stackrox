#!/bin/bash
# Post-job success handler — patches the winning copy's check-run to completed/success.
# Registered via gacts/run-and-post-run with post-if:success() so it only fires
# when the job succeeds. Env: GH_TOKEN, SPEC_REPO, SPEC_SHA.

check_id=$(cat /tmp/speedracer-check-id | head -1 | tr -d '[:space:]')
check_name=$(cat /tmp/speedracer-check-name | head -1 | tr -d '\n')
my_copy=$(cat /tmp/speedracer-copy | head -1 | tr -d '\n')

echo "Speedracer: copy '${my_copy}' won — marking check-run complete."

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
