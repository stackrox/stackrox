#!/usr/bin/env bash
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

kubectl apply -f "${DIR}/clairify.yaml"
