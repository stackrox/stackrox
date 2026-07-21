#!/usr/bin/env bash
# Builds a Slack incoming-webhook payload describing the current workflow
# run's failed jobs, failed steps, and error annotations.
#
# Usage: build-payload.sh <output-file>
#
# Environment: GITHUB_REPOSITORY, GITHUB_RUN_ID, GITHUB_WORKFLOW,
# GITHUB_EVENT_NAME, GITHUB_REF_NAME, GITHUB_SERVER_URL, MENTION,
# DESCRIPTION are required; INPUTS_JSON optionally carries the
# workflow_dispatch inputs as a JSON object. `gh` must be authenticated
# (GH_TOKEN) with actions:read and checks:read.

set -euo pipefail

if [ $# -ne 1 ]; then
    echo >&2 "usage: $0 <output-file>"
    exit 2
fi
output_file=$1

: "${GITHUB_REPOSITORY:?}" "${GITHUB_RUN_ID:?}" "${GITHUB_WORKFLOW:?}"
: "${GITHUB_EVENT_NAME:?}" "${GITHUB_REF_NAME:?}" "${GITHUB_SERVER_URL:?}"
: "${MENTION:?}" "${DESCRIPTION:?}"

max_jobs=10
max_annotations_per_job=5
max_annotation_chars=500

# per_page=100 covers every scanner workflow (largest matrix is well under
# 100 jobs), so pagination is intentionally omitted.
jobs_json=$(gh api "repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/jobs?filter=latest&per_page=100")

failed_jobs=$(jq '[.jobs[] | select(.conclusion == "failure") | {
    id,
    name,
    html_url,
    failed_step: ([.steps[]? | select(.conclusion == "failure") | .name][0] // null)
}]' <<<"$jobs_json")
total_failed=$(jq 'length' <<<"$failed_jobs")

enriched="[]"
while IFS= read -r job; do
    job_id=$(jq -r '.id' <<<"$job")
    annotations=$(gh api "repos/${GITHUB_REPOSITORY}/check-runs/${job_id}/annotations?per_page=100" |
        jq --argjson max_count "$max_annotations_per_job" \
            --argjson max_chars "$max_annotation_chars" '
            [.[]
             | select(.annotation_level == "failure")
             | .message
             | select(test("^Process completed with exit code") | not)
             | .[0:$max_chars]
            ][0:$max_count]')
    enriched=$(jq --argjson job "$job" --argjson annotations "$annotations" \
        '. + [$job + {annotations: $annotations}]' <<<"$enriched")
done < <(jq -c ".[0:${max_jobs}][]" <<<"$failed_jobs")

inputs_json=${INPUTS_JSON:-}
if [ -z "$inputs_json" ]; then
    inputs_json=null
fi

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

jq -n \
    --arg mention "$MENTION" \
    --arg workflow "$GITHUB_WORKFLOW" \
    --arg description "$DESCRIPTION" \
    --arg run_url "${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}" \
    --arg event "$GITHUB_EVENT_NAME" \
    --arg branch "$GITHUB_REF_NAME" \
    --argjson inputs "$inputs_json" \
    --argjson jobs "$enriched" \
    --argjson total_failed "$total_failed" \
    -f "${script_dir}/payload.jq" \
    >"$output_file"
