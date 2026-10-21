#!/usr/bin/env bash

is_nightly_run() {
    [[ "${BUILD_TAG:-}" =~ -nightly- ]] || [[ "${GITHUB_REF:-}" =~ nightly- ]]
}
