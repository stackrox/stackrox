#!/usr/bin/env bash

regex="log\.(Error|Debug|Info|Warn|Panic|Fatal)f*\(.*(query.Data).*\)"

tmpfile="$(mktemp)"
function delete_tmpfile() {
	rm -f "$tmpfile"
}
trap delete_tmpfile EXIT

git grep -inE "$regex" >"$tmpfile"

if [[ ! -s "$tmpfile" ]]; then
	exit 0
fi

echo >&2 "Found references to query.Data in log statements in the following files:"
cat >&2 "$tmpfile"

exit 1
