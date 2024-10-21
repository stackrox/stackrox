#!/usr/bin/env bash

set -euo pipefail

gate-version-compatibility-tests() {
  local exit_code="$1"

  if [[ "${exit_code}" == "0" ]]; then
    exit "${exit_code}"
  fi

  if [[ "${JOB_NAME}" == "" ]]; then
    exit "${exit_code}"
  fi

  if ! [[ "${JOB_NAME}" =~ .*-gke-version-compatibility-tests ]]; then
    exit "${exit_code}"
  fi

  local allow_list
  # Ensure JSON is formatted properly.
  allow_list=$(
    cat <<END_ALLOW_LIST
{
    "Central-v400.4": [
        {
            "test_name": "SummaryTest",
            "threshold": 30
        },
        {
            "test_name": "AutocompleteTest",
            "threshold": 5
        }
    ],
    "Central-v400.5": [
        {
            "test_name": "SummaryTest",
            "threshold": 15
        },
        {
            "test_name": "AutocompleteTest",
            "threshold": 5
        }
    ]
}
END_ALLOW_LIST
  )
  echo "===> Use allow list"
  echo "${allow_list}"

  local gate_for_versions=()
  while IFS='' read -r line; do gate_for_versions+=("$line"); done < <(echo "${allow_list}" | jq 'keys[]' --raw-output)
  echo "===> Gate for versions: ${gate_for_versions[*]}"

  local query
  # Query is defined without any placeholders or env variables.
  # This allows easier debugging, because it can be copy/pasted into/from BigQuery UI.
  query=$(
    cat <<END_OF_QUERY
WITH
  last_build_ids AS (
  SELECT
    BuildId,
    central_version
  FROM (
    SELECT
      BuildId,
      central_version,
      DENSE_RANK() OVER (PARTITION BY central_version ORDER BY Timestamp DESC) AS test_rank
    FROM (
      SELECT
        BuildId,
        Timestamp,
        REGEXP_EXTRACT(Name, r'Central-v\d{3}\.\d+') AS central_version,
      FROM
        acs-san-stackroxci.ci_metrics.stackrox_tests
      WHERE
        Classname = 'SummaryTest'
        AND Status IN ('failed',
          'passed')
        AND JobName = 'branch-ci-stackrox-stackrox-master-merge-gke-version-compatibility-tests' )
    WHERE
      central_version IS NOT NULL )
  WHERE
    test_rank <= 20
  GROUP BY
    BuildId,
    central_version
  ORDER BY
    central_version DESC )
SELECT
  t_all.Classname,
  t_all.central_version,
  t_all.total AS total_all,
  IFNULL(t_fail.total, 0) AS total_fail,
  CAST(ROUND((IFNULL(t_fail.total, 0) / IFNULL(t_all.total, 1))*100,0) AS INT64) AS fail_ratio
FROM (
  SELECT
    COUNT(*) AS total,
    Classname,
    central_version
  FROM
    acs-san-stackroxci.ci_metrics.stackrox_tests
  INNER JOIN
    last_build_ids lb_ids
  USING
    (BuildId)
  WHERE
    JobName = 'branch-ci-stackrox-stackrox-master-merge-gke-version-compatibility-tests'
    AND REGEXP_EXTRACT(Name, r'Central-v\d{3}\.\d+') = lb_ids.central_version
  GROUP BY
    Classname,
    central_version) t_all
LEFT JOIN (
  SELECT
    COUNT(*) AS total,
    Classname,
    central_version
  FROM
    acs-san-stackroxci.ci_metrics.stackrox_tests
  INNER JOIN
    last_build_ids lb_ids
  USING
    (BuildId)
  WHERE
    JobName = 'branch-ci-stackrox-stackrox-master-merge-gke-version-compatibility-tests'
    AND REGEXP_EXTRACT(Name, r'Central-v\d{3}\.\d+') = lb_ids.central_version
    AND Status IN ('failed')
  GROUP BY
    Classname,
    central_version) t_fail
ON
  t_all.Classname = t_fail.Classname
  AND t_all.central_version = t_fail.central_version
ORDER BY
  fail_ratio DESC;
END_OF_QUERY
  )

  echo "===> Fetching history for 'version-compatibility' tests"
  local res_rows_json
  res_rows_json=$(bq query --format=json --project_id="acs-san-stackroxci" --use_legacy_sql=false "${query}")

  echo "===> Got results from DB"
  echo "${res_rows_json}"

  local all_tests_in_db=()
  while IFS='' read -r line; do all_tests_in_db+=("$line"); done < <(echo "${res_rows_json}" | jq '[.[] | .Classname] | unique | .[]' --raw-output)

  echo "===> All tests in DB"
  echo "${all_tests_in_db[*]}"

  # Iterate over tests
  local total_all
  local fail_ratio
  local allowed_ratio
  for central_version in "${gate_for_versions[@]}"; do
    for test_name in "${all_tests_in_db[@]}"; do
      total_all=$(echo "${res_rows_json}" | jq '.[] | select( (.Classname=="'"${test_name}"'") and (.central_version=="'"${central_version}"'") ) | .total_all' --raw-output)
      fail_ratio=$(echo "${res_rows_json}" | jq '.[] | select( (.Classname=="'"${test_name}"'") and (.central_version=="'"${central_version}"'") ) | .fail_ratio' --raw-output)

      # If we don't have sufficient execution history, we will keep failing.
      if ((total_all < 20)); then
        echo "===> FAILED: no sufficient test history found for test '${test_name} -> ${central_version}'"
        exit "${exit_code}"
      fi

      # TODO: Is this a good decision?
      # TODO: The problem is that current failure is not taken into consideration. It would be better if we would
      # TODO: iterate over JUnit failures and check if specific failed test in the current run is flaky one.
      if ((fail_ratio == 0)); then
        continue
      fi

      allowed_ratio=$(echo "${allow_list}" | jq '."'"${central_version}"'"[] | select(.test_name=="'"${test_name}"'") | .threshold' --raw-output)
      if [[ "${allowed_ratio}" == "" ]]; then
        echo "===> FAILED: there is no allowed fail ratio defined for test '${test_name} -> ${central_version}' - (actual fail ratio: ${fail_ratio})"
        exit "${exit_code}"
      fi

      if ((allowed_ratio < fail_ratio)); then
        echo "===> FAILED: allowed fail ratio is below actual fail ratio for for test '${test_name} -> ${central_version}' - (${allowed_ratio} < ${fail_ratio})"
        exit "${exit_code}"
      fi
    done
  done

  echo "===> FLAKY: Test flakiness is within allowed boundaries. Returning success to CI pipeline"
  exit 0
}
