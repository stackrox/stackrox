#!/usr/bin/env bash

# This script is intended to be run in CircleCI as part of a workflow job.
# Collects stats for every job in the workflow that ends with '-tests' suffix,
#   and sends them to the pre-created GCP BigQuery table.
# The script expects all "-tests" jobs to be completed, otherwise it fails (see ./wait-for-jobs-completion.sh).

usage() {
  echo "Usage: $0 <GCP_PROJECT_ID> <BQ_DATASET_TABLE>"
  exit 2
}

[ -n "${CIRCLE_WORKFLOW_ID}" ] || { echo "Env var CIRCLE_WORKFLOW_ID not found"; exit 2; }
[ -n "${CIRCLE_TOKEN}" ] || { echo "Env var CIRCLE_TOKEN not found"; exit 2; }

if [[ -z "${CIRCLE_BRANCH}" && -z "${CIRCLE_TAG}" ]]; then
  echo "One of env vars CIRCLE_BRANCH or CIRCLE_TAG is expected to be set"
  exit 2
fi

GCP_PROJECT_ID=$1
BQ_DATASET_TABLE=$2

if [[ -z "${GCP_PROJECT_ID}" || -z "${BQ_DATASET_TABLE}" ]]; then
  usage
fi

WF_JOBS_DATA_URL="https://circleci.com/api/v2/workflow/${CIRCLE_WORKFLOW_ID}/job?circle-token=${CIRCLE_TOKEN}"

WF_JOBS_DATA=$(curl -s "${WF_JOBS_DATA_URL}")
message=$(echo "${WF_JOBS_DATA}" | jq -r '.message')
if [ "${message}" = "Workflow not found" ]; then
  echo >&2 "No workflow with ID ${CIRCLE_WORKFLOW_ID} was found."
  echo >&2 "It could be a CircleCI transient error, or circle-token isn't valid."
  exit 1
fi

echo "Collecting data from test jobs in the workflow ${CIRCLE_WORKFLOW_ID}."

WF_JOBS=$(echo "${WF_JOBS_DATA}" | jq '.items')
WF_JOBS_LENGTH=$(echo "${WF_JOBS}" | jq length)

QUERY_VALUES=()

BRANCH_VALUE=$([ -n "${CIRCLE_BRANCH}" ] && echo "'${CIRCLE_BRANCH}'" || echo "NULL")
TAG_VALUE=$([ -n "${CIRCLE_TAG}" ] && echo "'${CIRCLE_TAG}'" || echo "NULL")  

for (( i = 0; i < "${WF_JOBS_LENGTH}"; i++ )); do 
  data=$(echo "${WF_JOBS}" | jq ".[$i]")
  id=$(echo "${data}" | jq -r '.id')
  number=$(echo "${data}" | jq -r '.job_number // empty')
  name=$(echo "${data}" | jq -r '.name')
  status=$(echo "${data}" | jq -r '.status')
  started_at=$(echo "${data}" | jq '.started_at // empty')
  stopped_at=$(echo "${data}" | jq '.stopped_at // empty')

  if [[ "${name}" != *"-tests" ]]; then
    echo "Skipping job ${name} as it's not a test job (doesn't end with '-tests')."
    continue
  fi

  if [[ "${status}" != "success" && "${status}" != "failed" && "${status}" != "blocked" && "${status}" != "on_hold" ]]; then
    echo >&2 "It's expected that all test jobs are complete, yet found ${name} with status \"${status}\".";
    echo >&2 "Skipping stats collection for this incomplete job.";
    continue
  fi

  # (job_id, job_number, job_name, job_status, job_started_at, job_stopped_at, workflow_id, git_branch, git_tag)
  values="('${id}', ${number:-NULL}, '${name}', '${status}', ${started_at:-NULL}, ${stopped_at:-NULL}, '${CIRCLE_WORKFLOW_ID}', ${BRANCH_VALUE}, ${TAG_VALUE})"
  echo "BigQuery values for the job ${name}:"
  echo "  ${values}"

  QUERY_VALUES+=("${values}")
done

if [ "${#QUERY_VALUES[@]}" -eq 0 ]; then
  echo "No test jobs stats were collected."
  exit 0
fi

values=$(printf ",%s" "${QUERY_VALUES[@]}")
values="${values:1}"
query="INSERT ${BQ_DATASET_TABLE} (job_id, job_number, job_name, job_status, job_started_at, job_stopped_at, workflow_id, git_branch, git_tag)\
  VALUES ${values}"

echo "Executing query:"
echo "  ${query}"

bq query --headless --project_id="${GCP_PROJECT_ID}" --use_legacy_sql=false "${query}"
