#!/usr/bin/env bash
set -eou pipefail

elastic_username=$1
elastic_password=$2

source ./install-dependencies.sh
source setup-env-var.sh $elastic_username $elastic_password
./run-perf-tests.sh
