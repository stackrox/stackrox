#!/usr/bin/env bash
# Posts a Slack failure notification to $SLACK_WEBHOOK.
#
# Two modes:
#   - $MESSAGE is set: post that text verbatim (prefixed by $MENTION), no
#     GitHub API calls are made. Use this when the caller already has a
#     specific, curated message (e.g. a list of failed sources).
#   - $MESSAGE is unset: collect ::error:: annotations and per-job status for
#     the current workflow run via the GitHub API (optionally narrowed to
#     jobs whose name contains $JOB_NAME_FILTER, useful when this script runs
#     once per matrix leg of a reusable workflow), and build a message from that data.
#     If collection itself fails, a degraded but still-informative message is
#     sent instead of nothing at all.
#
# Required env vars: SLACK_WEBHOOK, RUN_URL, WORKFLOW, REPO
# Used only when $MESSAGE is unset: GH_TOKEN, RUN_ID
# Optional: MENTION, CONTEXT, MESSAGE, JOB_NAME_FILTER

set -euo pipefail

: "${SLACK_WEBHOOK:?SLACK_WEBHOOK is required}"
: "${RUN_URL:?RUN_URL is required}"
: "${WORKFLOW:?WORKFLOW is required}"
: "${REPO:?REPO is required}"

MENTION="${MENTION:-}"
CONTEXT="${CONTEXT:-}"
MESSAGE="${MESSAGE:-}"
JOB_NAME_FILTER="${JOB_NAME_FILTER:-}"

max_messages_per_job=10

# collect_annotations sets the global SUMMARY and JOB_STATUS variables from
# the current workflow run's jobs/annotations, optionally narrowed to job
# names containing $JOB_NAME_FILTER. Returns non-zero on any GitHub API
# failure.
collect_annotations() {
    : "${GH_TOKEN:?GH_TOKEN is required to collect annotations}"
    : "${RUN_ID:?RUN_ID is required to collect annotations}"

    # --paginate output is multiple JSON docs back-to-back; jq below handles
    # that as a stream, no --slurp needed.
    local jobs_json
    if ! jobs_json=$(gh api "repos/${REPO}/actions/runs/${RUN_ID}/jobs" --paginate); then
        return 1
    fi

    if [ -n "$JOB_NAME_FILTER" ]; then
        jobs_json=$(printf '%s' "$jobs_json" \
            | jq --arg filter "$JOB_NAME_FILTER" '{ jobs: [.jobs[] | select(.name | contains($filter))] }')
    fi

    # Plain text: rendered inside a Slack code block, not mrkdwn.
    SUMMARY=""
    while IFS=$'\t' read -r job_id job_name failed_step; do
        # Exclude GitHub's generic "Process completed..." annotation (noise).
        # --paginate: a job can have >30 annotations (API's page size).
        local messages
        if ! messages=$(gh api "repos/${REPO}/check-runs/${job_id}/annotations" --paginate \
            --jq '.[] | select(.annotation_level == "failure") | select(.message | test("^Process completed with exit code") | not) | .message'); then
            # Distinct fallback so an API failure isn't confused with "no annotations".
            messages="(failed to fetch annotations for this job via the GitHub API; see the workflow run directly for details)"
        fi

        # Blank line between job blocks, but not before the first one.
        if [ -n "$SUMMARY" ]; then
            SUMMARY+=$'\n\n'
        fi
        SUMMARY+="${job_name}: ${failed_step}"$'\n'
        if [ -n "$messages" ]; then
            local total
            total=$(printf '%s\n' "$messages" | grep -c . || true)
            SUMMARY+="$(printf '%s\n' "$messages" | sed -n "1,${max_messages_per_job}s/^/  - /p")"
            if [ "$total" -gt "$max_messages_per_job" ]; then
                SUMMARY+=$'\n'"  - ... and $((total - max_messages_per_job)) more (see the workflow run for full details)"
            fi
        else
            SUMMARY+="  - (no detailed error annotations found; see the workflow run for details)"
        fi
    done < <(printf '%s' "$jobs_json" | jq -r '.jobs[] | select(.conclusion | IN("failure","timed_out","action_required","neutral","stale","startup_failure")) | [.id, .name, ([.steps[]? | select(.conclusion | IN("failure","timed_out","action_required","neutral","stale","startup_failure")) | .name][0] // "unknown step")] | @tsv')

    if [ -z "$SUMMARY" ]; then
        SUMMARY="(no failed jobs found; see the workflow run for details)"
    fi

    # Safety-net truncation, on top of the per-job cap above.
    SUMMARY="${SUMMARY:0:3000}"

    # Status of every (filtered) job, icons matching GitHub's own.
    JOB_STATUS=""
    while IFS=$'\t' read -r job_name job_conclusion; do
        local icon
        case "$job_conclusion" in
            success) icon="✅" ;;
            skipped) icon="⏭️" ;;
            cancelled) icon="🚫" ;;
            in_progress) icon="🟡" ;;
            failure | timed_out | action_required | neutral | stale | startup_failure) icon="❌" ;;
            *) icon="❔" ;; # genuinely unrecognized future conclusion value
        esac
        JOB_STATUS+="${icon} ${job_name}"$'\n'
    done < <(printf '%s' "$jobs_json" | jq -r '.jobs[] | [.name, (.conclusion // "in_progress")] | @tsv')

    JOB_STATUS="${JOB_STATUS:0:3000}"
}

context_line=""
if [ -n "$CONTEXT" ]; then
    context_line="*Context:* ${CONTEXT}"$'\n'
fi

if [ -n "$MESSAGE" ]; then
    text="$MESSAGE"
elif collect_annotations; then
    # shellcheck disable=SC2016 # backticks below are a Slack code-block
    # delimiter, not shell command substitution.
    text=$(printf '*Failed Workflow:* <%s|%s>\n*Repository:* %s\n%s*Job Status:*\n```\n%s```\n*Failed steps:*\n```\n%s\n```' \
        "$RUN_URL" "$WORKFLOW" "$REPO" "$context_line" "$JOB_STATUS" "$SUMMARY")
else
    # Runs even if annotation collection failed, so a degraded message still
    # reaches on-call instead of nothing at all.
    text=$(printf ':warning: *Failed Workflow:* <%s|%s>\n*Repository:* %s\n%sCould not collect failure annotations for this run (the collection step itself failed) -- please check the workflow run directly for details.' \
        "$RUN_URL" "$WORKFLOW" "$REPO" "$context_line")
fi

if [ -n "$MENTION" ]; then
    text="${MENTION} ${text}"
fi

# --retry-all-errors means a slow-but-eventually-successful post could be
# retried and sent twice; unlikely in practice, but possible.
jq -n --arg text "$text" '{text: $text}' \
    | curl --fail --silent --show-error \
        --retry 3 --retry-all-errors --retry-delay 5 \
        --connect-timeout 10 --max-time 300 \
        -X POST -H 'Content-type: application/json' --data @- "$SLACK_WEBHOOK"
