#!/usr/bin/env bash

[[ -n "$CIRCLE_WORKFLOW_ID" ]] || {
	echo >&2 "No CircleCI workflow ID found. Is this job running on CircleCI?"
	exit 0
}

IFS=$'\n' read -d '' -r -a failed_steps < <(
	gsutil 2>/dev/null ls "gs://stackrox-ci-status/workflows/${CIRCLE_WORKFLOW_ID}/fatal-failures/**" |
	sed -E 's@^.*/@@g'
)

if [[ "${#failed_steps[@]}" == 0 ]]; then
	exit 0
fi

echo >&2 "Workflow $CIRCLE_WORKFLOW_ID is no longer live due to fatal errors in the following steps:"
printf >&2 " - %s\n" "${failed_steps[@]}"

exit 1
