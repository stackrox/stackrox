#!/usr/bin/env bash

# This script is intended to be run in CircleCI as part of a workflow job.
# It checks the status of other jobs in the workflow and waits for their completion.

[ -n "${CIRCLE_WORKFLOW_ID}" ] || { echo "Env var CIRCLE_WORKFLOW_ID not found"; exit 2; }
[ -n "${CIRCLE_JOB}" ] || { echo "Env var CIRCLE_JOB not found"; exit 2; }
[ -n "${CIRCLE_BUILD_NUM}" ] || { echo "Env var CIRCLE_BUILD_NUM not found"; exit 2; }
[ -n "${CIRCLE_TOKEN}" ] || { echo "Env var CIRCLE_TOKEN not found"; exit 2; }

WF_JOBS_DATA_URL="https://circleci.com/api/v2/workflow/${CIRCLE_WORKFLOW_ID}/job?circle-token=${CIRCLE_TOKEN}"

jobs_data=$(curl -s "${WF_JOBS_DATA_URL}")
message=$(echo "${jobs_data}" | jq -r '.message')
if [ "${message}" = "Workflow not found" ]; then
  echo >&2 "No workflow with ID ${CIRCLE_WORKFLOW_ID} was found."
  echo >&2 "It could be a CircleCI transient error, or circle-token isn't valid."
  exit 1
fi

jobs_length=$(echo "${jobs_data}" | jq '.items' | jq length)
if [ "${jobs_length}" -lt 2 ]; then
  echo "No other jobs found in this workflow, exiting..."
  exit 0
fi

echo "Found ${jobs_length} jobs in the workflow ${CIRCLE_WORKFLOW_ID}."
echo "Waiting for all jobs except ${CIRCLE_JOB} #${CIRCLE_BUILD_NUM} to finish."

WF_FINISHED=false
# iterate through all jobs in the workflow and flip the flag if all of them are complete
while [ "${WF_FINISHED}" = "false" ]; do
  WF_JOBS=$(curl -s "${WF_JOBS_DATA_URL}" | jq '.items')
  WF_JOBS_LENGTH=$(echo "${WF_JOBS}" | jq length)

  WF_FINISHED=true

  for (( i = 0; i < "${WF_JOBS_LENGTH}"; i++ )); do 
    job_data=$(echo "${WF_JOBS}" | jq ".[$i]")
    job_name=$(echo "${job_data}" | jq -r '.name')
    job_number=$(echo "${job_data}" | jq -r '.job_number')
    job_status=$(echo "${job_data}" | jq -r '.status')

    if [ "${job_number}" = "${CIRCLE_BUILD_NUM}" ]; then
      # skip if it's current job, we're waiting only for other jobs
      continue
    fi
      
    if [[ "${job_status}" != "success" && "${job_status}" != "failed" && "${job_status}" != "blocked" && "${job_status}" != "on_hold" ]]; then
      echo "Found at least one job running: ${job_name} #${job_number}."
      WF_FINISHED=false
    fi

    if [ "${WF_FINISHED}" = "false" ]; then
      # no reason to check the remaining jobs, we need to wait anyway
      break
    fi
  done

  if [ "${WF_FINISHED}" = "false" ]; then
    echo "Waiting for workflow jobs to finish, sleeping for 1 min..."
    sleep 60
  fi
done

echo "All other jobs complete!"
