#!/bin/bash
# shellcheck disable=SC2296,SC2009
# SC2009 : Consider using pgrep instead of grepping ps output
# SC2296 : Parameter expansions can't start with (

function join_by { local IFS="$1"; shift; echo "$*"; }
function path_prepend { PATH=$(join_by ':', "$1" "$PATH"); export PATH; }
function path_append { PATH=$(join_by ':', "$PATH" "$1"); export PATH; }

function info { echo "INFO: $*"; }
function warning { echo "WARNING: $*"; }
function error { >&2 echo "ERROR: $*"; exit 1; }

function bash_true { ((0 == 0)); }
function bash_false { ((0 == 1)); }
function bash_exit_success { >&1 echo "$@"; bash_true; exit $?; }
function bash_exit_failure { >&2 echo "$@"; bash_false; exit $?; }
function bash_return_success { bash_true; }
function bash_return_failure { bash_false; }
function errcho { >&2 echo "$*"; }

function cluster_safe_name { echo "$*" | sed -E 's/[^a-zA-Z0-9-]/-/g'; }
function hr { echo "--------"; }

function purge_dir {
  local dpath=$1
  rm -rf "$dpath"
  mkdir -p "$dpath"
  info "Created $dpath"
}

function shell_is_zsh { ps -ocommand -p"$$" | grep -q zsh; }
function shell_is_bash { ps -ocommand -p"$$" | grep -q bash; }
function is_linux { uname | grep -iq linux; }
function is_darwin { uname | grep -iq darwin; }

function symtab_lookup {
  ident="${1:-none}"

  if shell_is_zsh; then
    echo "${(P)ident}"
  elif shell_is_bash; then
    echo "${!ident}"
  fi
}

function var_match_string {
  local ident expected actual
  ident="${1:-FAKE_IDENT}"
  expected="${2:-}"
  actual=$(symtab_lookup "$ident")
  info "var_match_string($ident) [$expected] cmp [$actual]"
  [[ "$expected" == "$actual" ]]
}

function assert_file_exists {
  local path="$1"
  if ! [[ -e "$path" ]]; then
    error "file not found [$path]"
  fi
}

function cluster_is_openshift {
  kubectl config view | grep -q devshift-org
}
