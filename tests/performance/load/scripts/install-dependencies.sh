#!/usr/bin/env bash
set -eoux pipefail

git clone https://github.com/stackrox/stackrox.git
cd "${HOME}/stackrox"
git checkout jv-ROX-scripts-for-k6-load-testing
cd "${HOME}"
