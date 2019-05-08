#!/usr/bin/env bash

import_validate="$(dirname "${BASH_SOURCE[0]}")/import_validate.py"

# The following awk program parses lines (of the form `<file>:<line>:<message`) into
# fields delimited by colons, and groups the second fields (line numbers) keyed by
# the first field (file name). The grouped values are reduced by appending a `d` to
# each line number, and joining all those strings on `;` as the joining character.
# This creates output lines such as `/path/to/some/file.go 2d;7d`, where the second
# token is a sed command for deleting lines 2 and 7.
awk_prog='
{
  if ($3 == " Too many blank lines in imports") {
    lines[$1] = lines[$1] (lines[$1] == "" ? "" : ";") $2 "d"
  } else {
    print $0 >"/dev/stderr"
  }
} END {
  for (a in lines) print a, lines[a]
}
'

while IFS='' read -r line || [[ -n "$line" ]]; do
	if [[ "$line" =~ ^([^[:space:]]+)[[:space:]]+([^[:space:]]+)$ ]]; then
		file="${BASH_REMATCH[1]}"
		sed_cmd="${BASH_REMATCH[2]}"
	else
		echo >&2 "Malformed awk output line $line ..."
		exit 1
	fi
	deletions_only="${sed_cmd//[^d]/}"
	echo >&2 "Deleting ${#deletions_only} blank line(s) from $file"
	sed -i'.bak' -e "$sed_cmd" "$file"
	rm "${file}.bak"
done < <(awk -F: "$awk_prog" <("$import_validate" "$@") )
