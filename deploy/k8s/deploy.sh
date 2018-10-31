#!/usr/bin/env bash
set -e

K8S_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

bash "${K8S_DIR}/central.sh"
bash "${K8S_DIR}/sensor.sh"
