#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

export APOLLO_NO_REGISTRY_AUTH=true
$DIR/deploy.sh

