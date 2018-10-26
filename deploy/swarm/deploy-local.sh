#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd)"

export ROX_DISABLE_REGISTRY_AUTH=true
$DIR/deploy.sh

