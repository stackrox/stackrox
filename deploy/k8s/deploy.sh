#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

source "${K8S_DIR}"/central.sh
source "${K8S_DIR}"/sensor.sh
