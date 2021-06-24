#! /bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kubectl create -R -f "${DIR}/crds"
kubectl create -R -f "${DIR}/resources"
