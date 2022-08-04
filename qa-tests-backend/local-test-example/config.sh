#!/bin/bash

# Image was selected from a recent passing test run:
# https://app.circleci.com/pipelines/github/stackrox/stackrox/7557/workflows/3446e686-5b5e-4ab3-90a6-1c06a8201626/jobs/331096
# stackrox > Branch > master > Workflow > build_all > Job > build > Build Main Image
export MAIN_IMAGE_TAG="3.69.x-282-g91f0c3a24b"
export KUBECONFIG="/tmp/kubeconfig"
export STACKROX_SOURCE_ROOT="$GOPATH/src/github.com/stackrox/stackrox"
export WORKFLOW_SOURCE_ROOT="$GOPATH/src/github.com/stackrox/workflow"
export STACKROX_TEARDOWN_SCRIPT="$WORKFLOW_SOURCE_ROOT/bin/teardown"
export STACKROX_NAMESPACE="stackrox"
export CENTRAL_BUNDLE_DPATH="/tmp/central-bundle"
export QA_TESTS_BACKEND_DIR="$GOPATH/src/github.com/stackrox/stackrox/qa-tests-backend"
