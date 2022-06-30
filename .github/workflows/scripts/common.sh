#!/usr/bin/env bash

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
    for V in "$@"; do
        typeset -n VAR="$V"
        if [ -z "${VAR:-}" ]; then
            gh_log error "Variable $V is not set or empty"
            exit 1
        fi
    done
}
export -f check_not_empty

check_not_empty GITHUB_STEP_SUMMARY

# The output appears in the GitHub step summary box.
# Multiple lines are supported.
# Markdown is supported.
# Examples:
#     gh_summary "Markdown summary"
#     gh_summary <<EOF
#     Markdown summary
#     EOF
gh_summary() {
    if [ "$#" -eq 0 ]; then
        cat # for the data passed via the pipe
    else
        echo -e "$@" # for the data passed as arguments
    fi >>"$GITHUB_STEP_SUMMARY"
}
export -f gh_summary

if [ "$#" -gt 0 ]; then
    SCRIPT="$1"
    check_not_empty \
        SCRIPT \
        GITHUB_REPOSITORY \
        main_branch
    URL="/repos/$GITHUB_REPOSITORY/contents/.github/workflows/scripts/$SCRIPT.sh?ref=$main_branch"
    shift
    gh api -H "Accept: application/vnd.github.v3.raw" "$URL" | bash -s -- "$@"
fi
