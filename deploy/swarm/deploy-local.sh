#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

export PREVENT_DISABLE_REGISTRY_AUTH=true
export LOW_RESOURCES=true
$DIR/deploy.sh

