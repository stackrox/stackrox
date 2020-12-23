#!/usr/bin/env bash

# This script is intended to be run in CircleCI as part of a workflow job.
# Collects stats for every job in the workflow that ends with '-tests' suffix,
#   and sends them to the pre-created GCP BigQuery table.
# The script expects all "-tests" jobs to be completed, otherwise it fails (see ./wait-for-jobs-completion.sh).

usage() {
  echo "Usage: $0 <GCP_PROJECT_ID> <BQ_DATASET_TABLE> <BQ_TESTDATA_TABLE>"
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
BQ_TESTDATA_TABLE=$3

if [[ -z "${GCP_PROJECT_ID}" || -z "${BQ_DATASET_TABLE}" || -z "${BQ_TESTDATA_TABLE}" ]]; then
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
QUERY_FAILED_TEST_VALUES=()

BRANCH_VALUE=$([ -n "${CIRCLE_BRANCH}" ] && echo "'${CIRCLE_BRANCH}'" || echo "NULL")
TAG_VALUE=$([ -n "${CIRCLE_TAG}" ] && echo "'${CIRCLE_TAG}'" || echo "NULL")  

for (( job = 0; job < "${WF_JOBS_LENGTH}"; job++ )); do
  data=$(echo "${WF_JOBS}" | jq ".[$job]")
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

  if [[ "${status}" == "failed" ]]; then
    # fetch test details
    JOB_TEST_DETAILS_URL="https://circleci.com/api/v2/project/gh/stackrox/rox/${number}/tests?circle-token=${CIRCLE_TOKEN}"
    JOB_TEST_DETAILS=$(curl -s "${JOB_TEST_DETAILS_URL}")
    FAILED_TESTS=$(echo "${JOB_TEST_DETAILS}" | jq '[ .items[] | select( .result == "failure" ) ]')
    FAILED_TESTS_LENGTH=$(echo "${FAILED_TESTS}" | jq '. | length')
    for (( test = 0; test < "${FAILED_TESTS_LENGTH}"; test++ )); do
      failed_test=$(echo "${FAILED_TESTS}" | jq ".[$test]")
      test_name=$(echo "${failed_test}" | jq -r '.name' | sed "s/'/\'/g")
      test_classname=$(echo "${failed_test}" | jq -r '.classname')
      test_message=$(echo "${failed_test}" | jq -r '.message | gsub("[\\n\\t]"; "")' | cut -c -256 | sed 's/\\/\\\\/g' | sed "s/'//g")

      # (test_name, test_classname, test_message, test_started, job_number, job_name, workflow_id, git_branch, git_tag)
      test_values="(\"${test_name}\", '${test_classname}', '${test_message}', ${started_at:-NULL}, ${number:-NULL}, '${name}', '${CIRCLE_WORKFLOW_ID}', ${BRANCH_VALUE}, ${TAG_VALUE})"
      QUERY_FAILED_TEST_VALUES+=("${test_values}")
    done
  fi

  step_that_failed=""
  if [[ "${status}" == "failed" ]]; then
    # fetch step details
    JOB_STEP_DETAILS_URL="https://circleci.com/api/v1.1/project/gh/stackrox/rox/${number}?circle-token=${CIRCLE_TOKEN}"
    JOB_STEPS=$(curl -s "${JOB_STEP_DETAILS_URL}" | jq '.steps')
    if [[ "${JOB_STEPS}" != "null" ]]; then
      FIRST_FAILURE=$(echo "${JOB_STEPS}" | jq 'map(select( .actions[0].status != "success" and .actions[0].status != "canceled" ))[0]')
      if [[ "${FIRST_FAILURE}" != "null" ]]; then
        step_that_failed=$(echo "${FIRST_FAILURE}" | jq -r '.name' | sed "s/'/\'/g")
      fi
    fi
  fi

  # (job_id, job_number, job_name, job_status, job_started_at, job_stopped_at, workflow_id, git_branch, git_tag, step_that_failed)
  values="('${id}', ${number:-NULL}, '${name}', '${status}', ${started_at:-NULL}, ${stopped_at:-NULL}, '${CIRCLE_WORKFLOW_ID}', ${BRANCH_VALUE}, ${TAG_VALUE}, '${step_that_failed}')"
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
query="INSERT ${BQ_DATASET_TABLE} (job_id, job_number, job_name, job_status, job_started_at, job_stopped_at, workflow_id, git_branch, git_tag, step_that_failed)\
  VALUES ${values}"

echo "Executing query:"
echo "  ${query}"

bq query --headless --project_id="${GCP_PROJECT_ID}" --use_legacy_sql=false "${query}"

if [ "${#QUERY_FAILED_TEST_VALUES[@]}" -gt 0 ]; then
  echo "Some tests failed - updating failed test details"
  values=$(printf ",%s" "${QUERY_FAILED_TEST_VALUES[@]}")
  values="${values:1}"
  query="INSERT ${BQ_TESTDATA_TABLE} (test_name, test_classname, test_message, test_started, job_number, job_name, workflow_id, git_branch, git_tag)\
  VALUES ${values}"

  echo "Executing test details query:"
  echo "  ${query}"

  bq query --headless --project_id="${GCP_PROJECT_ID}" --use_legacy_sql=false "${query}"
fi
