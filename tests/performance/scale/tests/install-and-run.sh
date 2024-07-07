#!/usr/bin/env bash
set -eou pipefail

elastic_username=$1
elastic_password=$2
json_test_file=$3
cluster_name_prefix=$4
utilities_dir=$5
infra_token=$6

DIR="$(cd "$(dirname "$0")" && pwd)"

source "${DIR}"/install-dependencies.sh
source "${DIR}"/setup-env-var.sh "$elastic_username" "$elastic_password" "$infra_token"
"${DIR}"/run-perf-tests-2.sh "$json_test_file" "$cluster_name_prefix" "$utilities_dir"
