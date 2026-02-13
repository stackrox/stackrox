#!/usr/bin/env bash

tmpfile="$(mktemp)"
trap 'rm -f "${tmpfile}"' EXIT

git grep -E -o -h '(stackrox|scanner)-(build|test)-[0-9]+\.[0-9]+\.[0-9]' | grep -E -o '[0-9]+\.[0-9]+\.[0-9]' | sort -u >"$tmpfile"
git grep -E -o -h 'brew.registry.redhat.io/rh-osbs/openshift-golang-builder[^[:space:]]+' -- "**/konflux.Dockerfile" | sort -u >>"$tmpfile"

if [[ "$( wc -l < "$tmpfile" )" -eq 2 ]]
 then
	exit 0
fi

echo >&2 "Found multiple image versions:"
cat >&2 "$tmpfile"
echo >&2 "See $0 for the command which finds them."
exit 1
