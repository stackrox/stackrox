#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

export MITIGATE_DISABLE_REGISTRY_AUTH=true
export MITIGATE_DISABLE_DOCKER_TLS=true
$DIR/deploy.sh

