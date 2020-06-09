#!/bin/bash

# This script is intended to be run in CircleCI, and tells you whether or not the build is a nightly build
# The script exits with code
# 0 if the build IS a nightly build
# 1 if the build IS NOT a nightly build

usage() {
  echo "Usage: $0"
  exit 2
}

if [[ "${CIRCLE_TAG}" =~ .*-nightly-.* ]]; then
  exit 0
fi

exit 1
