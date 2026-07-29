#!/usr/bin/env bash
#
# Shared "--flag value" argument parser for the CVE fix verification
# scripts, factored out because determine-cve-scan-image-tag.sh,
# flatten-cve-scan-results.sh, post-cve-fix-comment.sh, and
# verify-cve-fix.sh each previously carried an identical copy of this loop.
#
# Usage (from a script that has already `set -euo pipefail`):
#   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
#   source "${SCRIPT_DIR}/lib/parse-flags.sh"
#   declare -A ARGS=()
#   parse_flags ARGS "$@"
#   FOO="${ARGS[foo]:-}"   # populated from a passed --foo value
#
# Every remaining argument must be a "--name value" pair; a flag given
# without a following value is stored as an empty string. Required-argument
# validation (checking which keys ended up non-empty) is left to the
# caller, since required flags differ per script.
#
# Exit codes:
#   2 - An argument didn't start with `--` (invalid usage).
#
parse_flags() {
  local -n out_args="$1"
  shift

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --*)
        if [[ $# -ge 2 && "$2" != --* ]]; then
          out_args["${1#--}"]="$2"
          shift 2
        else
          out_args["${1#--}"]=""
          shift
        fi
        ;;
      *)
        echo "$0: unknown argument: $1" >&2
        exit 2
        ;;
    esac
  done
}
