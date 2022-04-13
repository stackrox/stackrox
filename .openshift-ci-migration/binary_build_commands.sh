#!/usr/bin/env bash

set -euo pipefail

# These are not set in the binary_build_commands env.
export CI=true
export OPENSHIFT_CI=true

# For migration the build source needs to reside in /go/src/github.com/stackrox/rox-openshift-ci-mirror.
# The initial migrate.sh has cloned stackrox/stackrox and cloned the target branch. 
cp -ur /go/src/github.com/stackrox/stackrox/* /go/src/github.com/stackrox/rox-openshift-ci-mirror
