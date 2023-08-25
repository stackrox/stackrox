#!/usr/bin/env bash

set -euo pipefail

# A library of reusable bash functions

usage() {
    echo "lib.sh provides a library of reusable bash functions.
Invoke with:
  $ scripts/lib.sh method [args...]
Reuse with:
  . scripts/lib.sh
  method [args...]"
}

info() {
    echo "INFO: $(date): $*"
}

die() {
    echo >&2 "$@"
    exit 1
}

# Caution when editing: make sure groups would correspond to BASH_REMATCH use.
RELEASE_RC_TAG_BASH_REGEX='^([[:digit:]]+(\.[[:digit:]]+)*)(-rc\.[[:digit:]]+)?$'

roxcurl() {
  local url="$1"
  shift
  curl -sk -u "admin:${ROX_PASSWORD}" -k "https://${API_ENDPOINT}${url}" "$@"
}

is_release_version() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: is_release_version <version>"
    fi
    [[ "$1" =~ $RELEASE_RC_TAG_BASH_REGEX && -z "${BASH_REMATCH[3]}" ]]
}

is_RC_version() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: is_RC_version <version>"
    fi
    [[ "$1" =~ $RELEASE_RC_TAG_BASH_REGEX && -n "${BASH_REMATCH[3]}" ]]
}

# get_release_stream() - Gets the release major.minor from the output of `make tag`.
get_release_stream() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: get_release_stream <tag>"
    fi
    [[ "$1" =~ ^([[:digit:]]+\.[[:digit:]]+) ]]
    echo "${BASH_REMATCH[1]}"
}

# is_release_test_stream() - A major number of 0 is used to test release handling.
is_release_test_stream() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: is_release_test_stream <tag>"
    fi
    [[ "$1" =~ ^0\. ]]
}

is_CI() {
    [[ "${CI:-}" == "true" ]]
}

is_OPENSHIFT_CI() {
    [[ "${OPENSHIFT_CI:-}" == "true" ]]
}

is_GITHUB_ACTIONS() {
    [[ -n "${GITHUB_ACTION:-}" ]]
}

is_darwin() {
    uname -a | grep -i darwin >/dev/null 2>&1
}

is_linux() {
    uname -a | grep -i linux >/dev/null 2>&1
}

test_equals_non_silent() {
  if [[ "$#" -lt 2 ]]; then
    die "usage: test_equals_non_silent <arg1> <arg2>"
  fi

  if [[ "$1" != "$2" ]]; then
    die "Comparison failed: \"$1\" != \"$2\""
  fi
}

test_gt_non_silent() {
  if [[ "$#" -lt 2 ]]; then
    die "usage: test_gt_non_silent <arg1> <arg2>"
  fi

  if [[ "$1"  -le "$2" ]]; then
    die "Comparison failed: \"$1\" <= \"$2\""
  fi
}

test_empty_non_silent() {
  if [[ "$#" -lt 1 ]]; then
    die "usage: test_empty_non_silent <arg1>"
  fi

  if [[ -n "$1" ]]; then
    die "Comparison failed: \"$1\" is not empty"
  fi
}

require_environment() {
    if [[ "$#" -lt 1 ]]; then
        die "usage: require_environment NAME [reason]"
    fi

    (
        set +u
        if [[ -z "$(eval echo "\$$1")" ]]; then
            varname="$1"
            shift
            message="missing \"$varname\" environment variable"
            if [[ "$#" -gt 0 ]]; then
                message="$message: $*"
            fi
            die "$message"
        fi
    ) || exit 1
}

require_executable() {
    if [[ "$#" -lt 1 ]]; then
        die "usage: require_executable NAME [reason]"
    fi

    if ! command -v "$1" >/dev/null 2>&1; then
        varname="$1"
        shift
        message="missing \"$varname\" executable"
        if [[ "$#" -gt 0 ]]; then
            message="$message: $*"
        fi
        die "$message"
    fi
}

# retry() - retry a command up to a specific numer of times until it exits
# successfully, with exponential back off.
# (original source: https://gist.github.com/sj26/88e1c6584397bb7c13bd11108a579746)

retry() {
    if [[ "$#" -lt 3 ]]; then
        die "usage: retry <try count> <delay true|false> <command> <args...>"
    fi

    local tries=$1
    local delay=$2
    shift; shift;

    local count=0
    until "$@"; do
        exit=$?
        wait=$((2 ** count))
        count=$((count + 1))
        if [[ $count -lt $tries ]]; then
            info "Retry $count/$tries exited $exit"
            if $delay; then
                info "Retrying in $wait seconds..."
                sleep $wait
            fi
            if [[ -n "${RETRY_HOOK:-}" ]]; then
                $RETRY_HOOK
            fi
        else
            echo "Retry $count/$tries exited $exit, no more retries left."
            return $exit
        fi
    done
    return 0
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        usage
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
