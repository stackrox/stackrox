#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

TOOL="$(basename "$0" ".sh" | sed -E 's/^go-//g')"

set -e

die() {
	echo >&2 "$@"
	exit 1
}

RACE="${RACE:-false}"
REPO_ROOT="${SCRIPT_DIR}/.."

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
done < <(cd "${REPO_ROOT}"; ./status.sh)

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
		go_package="$(cd "${REPO_ROOT}"; go list -e "./$(dirname "$go_file")")"

		x_defs+=(-X "\"${go_package}.${go_var}=${!varname}\"")
	fi
done < <(git -C "${REPO_ROOT}" grep -n '//XDef:' -- '*.go')
if [[ "${#x_def_errors[@]}" -gt 0 ]]; then
	printf >&2 "%s\n" "${x_def_errors[@]}"
	exit 1
fi

# Build ldflags: full version for builds, stable base tag for tests.
# Go includes -X ldflags in the link ActionID (exec.go:1404), so per-commit
# values (MainVersion, GitShortSha) invalidate the test result cache.
# Using a stable base tag for tests keeps test ActionIDs constant across commits.
build_ldflags=("${x_defs[@]}")

BASE_VERSION="$(cd "${REPO_ROOT}"; git describe --tags --abbrev=0 --exclude '*-nightly-*' 2>/dev/null || echo "")"
if [[ -n "$BASE_VERSION" ]]; then
  version_pkg="$(cd "${REPO_ROOT}"; go list -e ./pkg/version/internal)"
  test_ldflags=()
  # x_defs contains pairs: -X "pkg.Var=value". Iterate in steps of 2.
  for (( i=0; i<${#x_defs[@]}; i+=2 )); do
    xflag="${x_defs[i]}"      # -X
    xval="${x_defs[i+1]}"     # "pkg.Var=value"
    if [[ "$xval" == *"MainVersion="* ]]; then
      test_ldflags+=(-X "\"${version_pkg}.MainVersion=${BASE_VERSION}\"")
    elif [[ "$xval" == *"GitShortSha="* ]]; then
      # Skip GitShortSha for tests — not needed and changes per commit.
      :
    else
      test_ldflags+=("$xflag" "$xval")
    fi
  done
else
  # No git tags available — fall back to full ldflags for tests too.
  test_ldflags=("${x_defs[@]}")
fi

if [[ "$DEBUG_BUILD" != "yes" ]]; then
  build_ldflags+=(-s -w)
  test_ldflags+=(-s -w)
fi

if [[ "${CGO_ENABLED}" != 0 ]]; then
  echo >&2 "CGO_ENABLED is not 0. Compiling with -linkmode=external"
  build_ldflags+=('-linkmode=external')
  test_ldflags+=('-linkmode=external')
fi

function invoke_go() {
  local tool="$1"
  shift
  local flags
  if [[ "$tool" == "test" ]]; then
    flags="${test_ldflags[*]}"
  else
    flags="${build_ldflags[*]}"
  fi
  if [[ "$RACE" == "true" ]]; then
    CGO_ENABLED=1 go "$tool" -race -buildvcs=false -ldflags="${flags}" -tags "$(tr , ' ' <<<"$GOTAGS")" "$@"
  else
    go "$tool" -buildvcs=false -ldflags="${flags}" -tags "$(tr , ' ' <<<"$GOTAGS")" "$@"
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
