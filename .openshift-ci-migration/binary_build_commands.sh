#!/usr/bin/env bash

set -euo pipefail

# These are not set in the binary_build_commands env.
export CI=true
export OPENSHIFT_CI=true
