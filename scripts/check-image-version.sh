#!/usr/bin/env bash

# This script is used to check that all the test and build images are in the same version.

tmpfile="$(mktemp)"
trap 'rm -f "${tmpfile}"' EXIT

git grep -E -o -h '(stackrox|scanner)-(build|test|ui-test)-[0-9]+\.[0-9]+\.[0-9]+(-[0-9]+-g[0-9a-f]+)?' | grep -E -o '[0-9]+\.[0-9]+\.[0-9]+(-[0-9]+-g[0-9a-f]+)?' | sort -u >"$tmpfile"

if [[ "$( wc -w < "$tmpfile" )" -eq 1 ]]
 then
	exit 0
fi

echo >&2 "## Found multiple image versions:"
cat >&2 "$tmpfile"
echo >&2 "## See $0 for the command which finds them."
exit 1
