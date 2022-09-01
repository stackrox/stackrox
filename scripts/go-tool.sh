#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

TOOL="$(basename "$0" ".sh" | sed -E 's/^go-//g')"

set -e

die() {
	echo >&2 "$@"
	exit 1
}

RACE="${RACE:-false}"

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
done < <(cd "${SCRIPT_DIR}/.."; ./status.sh)

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
		go_package="$(cd "${SCRIPT_DIR}/.."; go list -e "./$(dirname "$go_file")")"

		x_defs+=(-X "\"${go_package}.${go_var}=${!varname}\"")
	fi
done < <(git -C "${SCRIPT_DIR}/.." grep -n '//XDef:' -- '*.go')
if [[ "${#x_def_errors[@]}" -gt 0 ]]; then
	printf >&2 "%s\n" "${x_def_errors[@]}"
	exit 1
fi

ldflags=("${x_defs[@]}")
if [[ "$DEBUG_BUILD" != "yes" ]]; then
  ldflags+=(-s -w)
fi

if [[ "${CGO_ENABLED}" != 0 ]]; then
  echo >&2 "CGO_ENABLED is not 0. Compiling with -linkmode=external"
  ldflags+=('-linkmode=external')
fi

function invoke_go() {
  tool="$1"
  shift
  if [[ "$RACE" == "true" ]]; then
    CGO_ENABLED=1 go "$tool" -race -ldflags="${ldflags[*]}" -tags "$(tr , ' ' <<<"$GOTAGS")" "$@"
  else
    go "$tool" -ldflags="${ldflags[*]}" -tags "$(tr , ' ' <<<"$GOTAGS")" "$@"
  fi
}

function go_build() (
  export GOOS="${GOOS:-${DEFAULT_GOOS}}"
  [[ -n "$GOOS" ]] || die "GOOS must be set"

  for main_srcdir in "$@"; do
    if ! [[ "${main_srcdir}" =~ ^\.?/ ]]; then
      main_srcdir="./${main_srcdir}"
    fi
    bin_name="$(basename "$main_srcdir")"
    output_file="bin/${GOOS}/${bin_name}"
    if [[ "$GOOS" == "windows" ]]; then
      output_file="${output_file}.exe"
    fi
    mkdir -p "$(dirname "$output_file")"
    echo >&2 "Compiling Go source in ${main_srcdir} to ${output_file}"
    invoke_go_build -o "$output_file" "$main_srcdir"
  done
)

function go_build_file() {
    local input_file="$1"
    local output_file="$2"
    invoke_go_build -o "${output_file}" "${input_file}"
}

function invoke_go_build() {
  local gcflags=()
  if [[ "$DEBUG_BUILD" == "yes" ]]; then
    gcflags+=(-gcflags "all=-N -l")
  fi
  invoke_go build -trimpath "${gcflags[@]}" "$@"
}

function go_run() (
  invoke_go run "$@"
)

function go_test() (
  unset GOOS
  invoke_go test "$@"
)

case "$TOOL" in
  build)
    go_build "$@"
    ;;
  build-file)
    go_build_file "$1" "$2"
    ;;
  test)
    go_test "$@"
    ;;
  run)
    go_run "$@"
    ;;
  *)
    die "Unknown go tool '${TOOL}'"
    ;;
esac
