#!/usr/bin/env bash
artifacts_dir="${TEST_RESULTS_OUTPUT_DIR:-cypress/test-results}/artifacts"
export CYPRESS_VIDEOS_FOLDER="${artifacts_dir}/videos"
export CYPRESS_SCREENSHOTS_FOLDER="${artifacts_dir}/screenshots"

mkdir -p "$artifacts_dir" && touch "${artifacts_dir:-/tmp}/cypress-err.txt"
CYPRESS_INTERNAL_BROWSER_CONNECT_TIMEOUT=180000 DEBUG="cypress*" NO_COLOR=1 cypress "$@" 2> "${artifacts_dir:-/tmp}/cypress-err.txt"
