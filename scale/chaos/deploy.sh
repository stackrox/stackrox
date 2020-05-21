#!/bin/bash

set -euo pipefail

dir="$(dirname "${BASH_SOURCE[0]}")"

export TAG="$(git describe --tags --abbrev=10 --dirty --long)"

envsubst < "${dir}/deploy.yaml" | kubectl apply -f -
