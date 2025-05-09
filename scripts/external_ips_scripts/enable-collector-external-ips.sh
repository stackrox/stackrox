#!/usr/bin/env bash
set -eou pipefail

kubectl create -f collector-config-enabled.yml
