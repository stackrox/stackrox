#!/usr/bin/env bash
set -eo pipefail

# Wrapper around helm to sweeten the development experience by rendering both Helm charts before executing Helm
# Example testing central-services-chart:
# ./debug-helm-chart.sh upgrade --install --dry-run stackrox-central-services ./stackrox-central-services-chart -n stackrox --set imagePullSecrets.allowNone=true
#
# Usage: ./debug-helm-chart.sh [NAME] [CHART] [flags]

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Set $IMAGE_DEFAULTS to change the image defaults. Use opensource to use the stackrox-io images.
image_defaults=${IMAGE_DEFAULTS:-"development_build"}

"$DIR/roxctl.sh" helm output central-services --image-defaults="$image_defaults" --remove --debug
"$DIR/roxctl.sh" helm output secured-cluster-services --image-defaults="$image_defaults" --remove --debug

helm "$@"
