#!/usr/bin/env bash

tmpfile="$(mktemp)"
trap 'rm -f "${tmpfile}"' EXIT

echo "Checking stackrox-ci image..."
git grep -E -o -h '(stackrox|scanner)-(build|test)-[0-9]+\.[0-9]+\.[0-9]' | grep -E -o '[0-9]+\.[0-9]+\.[0-9]' | sort -u | tee /dev/fd/2 >"$tmpfile"
count=$(wc -l <"$tmpfile")
if [[ "$count" -ne 1 ]]; then
  >&2 echo "Did not find a single version of the stackrox-ci image. See the output above"
  exit 6
fi

echo "Checking Go builder..."
git grep -E -o -h 'brew.registry.redhat.io/rh-osbs/openshift-golang-builder[^[:space:]]+' -- "**/konflux*.Dockerfile" | sort -u | tee /dev/fd/2 >"$tmpfile"
count=$(wc -l <"$tmpfile")
if [[ "$count" -ne 1 ]]; then
  >&2 echo "Did not find a single version of the Go builder image. See the output above"
  exit 6
fi
