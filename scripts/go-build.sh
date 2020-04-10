#!/usr/bin/env bash

set -e

die() {
	echo >&2 "$@"
	exit 1
}

RACE="${RACE:-false}"
export GOOS="${GOOS:-${DEFAULT_GOOS}}"
[[ -n "$GOOS" ]] || die "GOOS must be set"

x_defs=()
x_def_errors=()

while read -r line || [[ -n "$line" ]]; do
	if [[ "$line" =~ ^[[:space:]]*$ ]]; then
		continue
	elif [[ "$line" =~ ^([^[:space:]]+)[[:space:]]+(.*)[[:space:]]*$ ]]; then
		var="${BASH_REMATCH[1]}"
		def="${BASH_REMATCH[2]}"
		eval "status_${var}=$(printf '%q' "$def")"
	else
		die "Malformed status.sh output line ${line}"
	fi
done < <(./status.sh)

while read -r line || [[ -n "$line" ]]; do
	if [[ "$line" =~ ^[[:space:]]*$ ]]; then
		continue
	elif [[ "$line" =~ ^([^:]+):([[:digit:]]+):[[:space:]]*(var[[:space:]]+)?([^[:space:]]+)[[:space:]].*//XDef:([^[:space:]]+)[[:space:]]*$ ]]; then
		go_file="${BASH_REMATCH[1]}"
		go_line="${BASH_REMATCH[2]}"
		go_var="${BASH_REMATCH[4]}"
		status_var="${BASH_REMATCH[5]}"

		varname="status_${status_var}"
		[[ -n "${!varname}" ]] || x_def_errors+=(
			"Variable ${go_var} defined in ${go_file}:${go_line} references status var ${status_var} that is not part of the status.sh output"
		)
		go_package="$(go list -e ./$(dirname "$go_file"))"

		x_defs+=(-X "\"${go_package}.${go_var}=${!varname}\"")
	fi
done < <(git grep -n '//XDef:' -- '*.go')
if [[ "${#x_def_errors[@]}" -gt 0 ]]; then
	printf >&2 "%s\n" "${x_def_errors[@]}"
	exit 1
fi

ldflags=(-s -w "${x_defs[@]}")

go_build() {
	main_srcdir="$1"
	if ! [[ "${main_srcdir}" =~ ^\.?/ ]]; then
		main_srcdir="./${main_srcdir}"
	fi
	bin_name="$(basename "$main_srcdir")"
	if [[ "$RACE" == "true" ]]; then
	  bin_name="${bin_name}-race"
	fi
	output_file="bin/${GOOS}/${bin_name}"
	if [[ "$GOOS" == "windows" ]]; then
		output_file="${output_file}.exe"
	fi
	mkdir -p "$(dirname "$output_file")"
	echo >&2 "Compiling Go source in ${main_srcdir} to ${output_file}"
	if [[ "$RACE" == "true" ]]; then
	  CGO_ENABLED=1 go build -race -ldflags="${ldflags[*]}" -tags "$(tr , ' ' <<<"$GOTAGS")" -o "$output_file" "$main_srcdir"
	else
	  go build -ldflags="${ldflags[*]}" -tags "$(tr , ' ' <<<"$GOTAGS")" -o "$output_file" "$main_srcdir"
	fi
}

for arg in "$@"; do
	go_build "$arg"
done
