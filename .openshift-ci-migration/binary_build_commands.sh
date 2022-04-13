#!/usr/bin/env bash

set -euo pipefail

# For migration the build source needs to reside in /go/src/github.com/stackrox/rox-openshift-ci-mirror.
# The initial migrate.sh has cloned stackrox/stackrox and cloned the target branch. 
cp -ur /go/src/github.com/stackrox/stackrox/* /go/src/github.com/stackrox/rox-openshift-ci-mirror
