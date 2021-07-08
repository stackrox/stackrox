#!/bin/bash

set -euo pipefail

dir="$(dirname "${BASH_SOURCE[0]}")"

export TAG="$(make --quiet -C "${dir}/.." tag)"

envsubst < "${dir}/deploy.yaml" | kubectl apply -f -
