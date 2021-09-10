#!/usr/bin/env bash
set -eo pipefail

# Wrapper around helm but builds roxctl and generates new rox charts.
# Example testing central-services-chart:
# ./debug-helm-chart.sh upgrade --install --dry-run stackrox-central-services ./stackrox-central-services-chart -n stackrox --set imagePullSecrets.allowNone=true
#
# Usage: ./debug-helm-chart.sh [NAME] [CHART] [flags]

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [[ "$(uname)" == "Darwin"* ]]; then
  make -C "$DIR/../" cli-darwin
else
  make -C "$DIR/../" cli-linux
fi

"$DIR/roxctl.sh" helm output central-services --remove --debug
"$DIR/roxctl.sh" helm output secured-cluster-services --remove --debug

helm $@
