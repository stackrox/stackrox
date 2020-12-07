#!/usr/bin/env bash
set -eo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

source "${DIR}"/../../deploy/common/k8sbased.sh

if [[ -z "$1" ]]; then
  echo "Expected component as the first arg"
  echo "Available [sensor, central]"
  exit 1
fi

component="$1"
case "${component}" in
"sensor")
  hotload_binary kubernetes-sensor kubernetes sensor
  ;;
"central")
  hotload_binary central central central
  ;;
*)
  echo "Invalid input: ${component}"
  echo "Available [sensor, central]"
  exit 1
  ;;
esac
