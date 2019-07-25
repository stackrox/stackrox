#!/bin/bash

set -e

kubectl create -f scripts/ci/psp/psp.yaml
