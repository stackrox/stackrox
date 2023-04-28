#!/usr/bin/env bash
set -eo pipefail

# Mount local binaries to enable HOTRELOAD after a deployment.
# It helps to run recently build binaries (i.e. from `make fast-central`) inside the cluster by
# only deleting the pod, instead of building a new main image.
# Usage: ./enable-hotreload.sh [sensor,central,migrator,admission]

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# shellcheck source=../deploy/common/k8sbased.sh
source "${DIR}"/../deploy/common/k8sbased.sh

if [[ -z "$1" ]]; then
  echo "Expected component as the first arg"
  echo "Available [sensor, central, migrator, admission]"
  exit 1
fi

component="$1"
case "${component}" in
"sensor")
  hotload_binary bin/kubernetes-sensor kubernetes sensor
  ;;
"central")
  hotload_binary central central central
  ;;
"migrator")
  hotload_binary bin/migrator migrator central
  ;;
"admission"|"admission-control"|"admission-controller")
  hotload_binary admission-control admission-control admission-control
  ;;
*)
  echo "Invalid input: ${component}"
  echo "Available [sensor, central, migrator, admission]"
  exit 1
  ;;
esac
