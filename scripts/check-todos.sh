#!/usr/bin/env bash

function join_str() {
	sep="$1"
	shift
	printf "${sep}%s" "$@" | tail -c +$((${#sep} + 1))
	echo
}

regex="todo\([^\)]*\b($(join_str "|" "$@"))\b"

tmpfile="$(mktemp)"
function delete_tmpfile() {
	rm -f "$tmpfile"
}
trap delete_tmpfile EXIT

git grep -inE "$regex" >"$tmpfile"

if [[ ! -s "$tmpfile" ]]; then
	exit 0
fi

echo >&2 "Found TODO references to $(join_str ", " "$@") in the following files:"
cat >&2 "$tmpfile"

exit 1
