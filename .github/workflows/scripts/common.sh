#!/bin/bash

# The output appears in a dedicated emphasized GitHub step box.
# Multiple lines are not supported.
# Markdown is not supported.
gh_log() {
    local LEVEL="$1"
    shift
    echo "::$LEVEL::$*"
}
export -f gh_log

# Sets a step output value.
# Multiple lines are not supported: join with URL escaping.
gh_output() {
    local NAME="$1"
    shift
    echo "::set-output name=$NAME::$*"
}
export -f gh_output

check_not_empty() {
    local VAR
    typeset -n VAR="$1"
    if [ -z "${VAR:-}" ]; then
        gh_log error "Variable $1 is not set or empty"
        exit 1
    fi
}
export -f check_not_empty

check_not_empty GITHUB_STEP_SUMMARY

# The output appears in the GitHub step summary box.
# Multiple lines are supported.
# Markdown is supported.
gh_summary() {
    if [ "$#" -eq 0 ]; then
        cat
    else
        echo -e "$@"
    fi >>"$GITHUB_STEP_SUMMARY"
}
export -f gh_summary
