#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

TOOL="$(basename "$0" ".sh" | sed -E 's/^go-//g')"

set -e

die() {
	echo >&2 "$@"
	exit 1
}

RACE="${RACE:-false}"

# Generate version file instead of using ldflags for version info.
# This improves Go build cache efficiency - only pkg/version/internal
# needs recompilation when version changes, not the entire dependency graph.
"${SCRIPT_DIR}/generate-version.sh"

ldflags=()
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
  [[ -n "$GOARCH" ]] || die "GOARCH must be set"

  dirs=()
  for main_srcdir in "$@"; do
    if ! [[ "$main_srcdir" =~ ^\.?/ ]]; then
      main_srcdir="./$main_srcdir"
    fi
    dirs+=("$main_srcdir")
  done

  output="bin/${GOOS}_${GOARCH}"
  mkdir -p "$output"

  echo >&2 "Compiling Go source in ${dirs[*]} to ${output}"
  invoke_go_build -o "$output" "${dirs[@]}"
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
