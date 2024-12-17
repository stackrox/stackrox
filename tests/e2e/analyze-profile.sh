#!/usr/bin/env bash

set -euo pipefail

FILE="${1:-}"
BASE="${2:-}"

function usage() {
  printf "Usage: %s <PROFILE_CURRENT_PR> [PROFILE_REFERENCE_MASTER]\n" "$0"
}

if ! test -f "$FILE"; then
  usage
  exit 1
fi

NUM_TOP=5

if ! command -v go > /dev/null; then
  echo "Cannot work without 'go' binary"
  exit 2
fi

function main() {
  declare -a topBase
  declare -a topFile

  # Read command to array
  IFS=$'\n' read -r -d '' -a topFile < <( getTopEntries "$FILE" && printf '\0' )
  printSummary "$FILE" "Top $NUM_TOP memory consumers in this PR (file: '$FILE'):" "${topFile[@]}"

  if [[ -n "$BASE" ]]; then
    IFS=$'\n' read -r -d '' -a topBase < <( getTopEntries "$BASE" && printf '\0' )
    echo
    printSummary "$BASE" "Top $NUM_TOP memory consumers in the reference branch (file: '$BASE'):" "${topBase[@]}"
    echo
    printDiff "$FILE" "$BASE" "Diff of top $NUM_TOP memory consumers between the reference branch (file: '$BASE') and the current PR (file: '$FILE'). Negative value means lower memory usage for current PR:"
  fi
}

function getTopEntries() {
  local file
  file="$1"
  ARGS=("-sample_index=alloc_space" "-top" "-flat" "-nodecount" "$NUM_TOP")
  go tool pprof "${ARGS[@]}" "$file" | grep -v 'flat%' | grep -v 'Showing nodes' | grep '%' | awk '{print $6}'
}

function getFlatValue() {
  local file
  file="$1"
  local entry
  entry="$2"
  go tool pprof "-sample_index=alloc_space" "-top" "-flat" "-show=$entry" "$file" | grep -v 'flat%' | grep -v 'Showing nodes' | grep '%' | awk '{print $1 " " $5}'
}

function printSummary() {
  local file
  file="$1"; shift
  local heading
  heading="$1"; shift
  arr=("$@")

  f1="$(mktemp)"
  echo "Memory-consumed %-of-total-memory Package" >> "$f1"
  for key in "${arr[@]}"
  do
    qkey="$(printf "%q" "$key")"
    value="$(getFlatValue "$file" "$qkey")"
    printf "\t%s\t%s\n" "$value" "$key" >> "$f1"
  done

  echo "$heading"
  echo '```'
  column -t "$f1"
  echo '```'
}

function printDiff() {
  local file
  file="$1"
  local ref
  ref="$2"
  local heading
  heading="$3"

  f1="$(mktemp)"
  echo "Max-mem-allocation-diff Max-diff-% Cumulative-allocations-diff Cumulative-diff-% pkg" >> "$f1"
  go tool pprof "-sample_index=alloc_space" "-top" "-flat" "-nodecount" "$NUM_TOP" "-diff_base=$ref" "$file" | grep -v 'Showing nodes' | grep -v 'flat%' | grep '%' >> "$f1"
  echo "$heading"
  echo '```'
  column -t "$f1"
  echo '```'
}

main "${@}"
