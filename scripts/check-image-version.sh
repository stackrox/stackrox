#!/usr/bin/env bash

tmpfile="$(mktemp)"
trap 'rm -f "${tmpfile}"' EXIT

git grep -E -o -h '(stackrox|scanner)-(build|test)-0.[0-9]+\.[0-9]' | grep -E -o '0.[0-9]+\.[0-9]' | sort -u >"$tmpfile"

if [[ "$( wc -w < "$tmpfile" )" -eq 1 ]]
 then
	exit 0
fi

echo >&2 "Found multiple image versions:"
cat >&2 "$tmpfile"

exit 1
