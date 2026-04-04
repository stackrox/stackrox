#!/usr/bin/env bash

RUNNER_TOKEN=$(gh api repos/davdhacs/stackrox-reduced/actions/runners/registration-token -X POST --jq '.token')
echo $RUNNER_TOKEN

docker run -it --rm \
  --platform linux/amd64 \
  -e DISABLE_AUTO_UPDATE=true \
    -e RUNNER_NAME=colima-runner \
    -e RUNNER_WORKDIR=/tmp/runner-work \
    -e RUNNER_REPOSITORY_URL=https://github.com/davdhacs/stackrox-reduced \
    -e REPO_URL=https://github.com/davdhacs/stackrox-reduced \
    -e RUNNER_TOKEN="$RUNNER_TOKEN" \
    -e RUNNER_REUSE=true \
    -e LABELS=self-hosted \
    myoung34/github-runner:latest
