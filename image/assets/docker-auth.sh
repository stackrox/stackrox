#!/usr/bin/env bash

# Copyright (c) 2018 StackRox Inc.
#
# Reads Docker credentials from ~/.docker/config.json / credentials store / terminal prompt, and outputs them as
# a base64 encoded auth token, or an entire docker auths config (if `-m k8s` is specified).

b64enc_cmd=()

if type openssl >/dev/null 2>&1; then
	b64enc_cmd=(openssl base64)
elif type base64 >/dev/null 2>&1; then
	b64enc_cmd=(base64)
fi

if ! "${b64enc_cmd[@]}" <<<"" >/dev/null 2>&1; then
	echo >&2 "No usable base64 command was found on your system!"
	exit 1
fi

function b64enc() {
	echo -n "$1" | "${b64enc_cmd[@]}" | tr -d '\n'
}

function url2std() {
	tr '_-' '/+' | tr -d '\n'
}

function std2url() {
	tr '/+' '_-' | tr -d '\n'
}

output_mode=""
registry_url=""

while [[ $# > 0 ]]; do
	case "$1" in
	-m)
		shift
		output_mode="$1"
		;;
	-*)
		echo >&2 "Invalid option '$1'"
		exit 1
		;;
	*)
		[[ -z "$registry_url" ]] || {
			echo >&2 "Exactly one registry must be specified."
			exit 1
		}
		registry_url="$1"
		;;
	esac
	shift
done

if [[ -z "$registry_url" ]]; then
	echo >&2 "Usage: $0 [-m <output mode>] <registry url>"
	exit 1
fi

if [[ ! -p /dev/stdout ]]; then
	echo >&2 "For security reasons, output will only be written to a pipe"
	exit 1
fi

if [[ -n "$output_mode" && "$output_mode" != "k8s" ]]; then
	echo >&2 "Invalid output mode '${output_mode}'"
	exit 1
fi

username="${REGISTRY_USERNAME}"
password="${REGISTRY_PASSWORD}"

function print_auth() {
	local auth_token="$1"
	if [[ -z "$output_mode" ]]; then
		echo "$auth_token"
		exit 0
	fi
	if [[ "$output_mode" == "k8s" ]]; then
		local auth_token_std="$(url2std <<<"$auth_token")"
		local auths_str="{\"auths\":{\"$registry_url\":{\"auth\":\"${auth_token_std}\"}}}"
		b64enc "$auths_str"
	fi
    exit 0
}

function mkauth() {
	local username="$1"
	local password="$2"

	b64enc "${username}:${password}" | std2url
}

function try_dockercfg_plain() {
	local components=()
	local dockercfg="$1"
    IFS=$'\n' read -d '' -r -a components < <(
        jq -r <<<"$dockercfg" '.auths["'"${registry_url}"'"] | (.auth // "", .username // "", .password // "")')
    local auth_str="${components[0]}"
    if [[ -n "$auth_str" ]]; then
        echo >&2 "Using authentication token for ${registry_url} from ~/.docker/config.json."
        print_auth "$auth_str"
    fi
    [[ -z "$username" || "$username" == "${components[1]}" ]] || return
    # stackrox.io returns a refresh token instead of a username and password so we should fall back to
    # user input username and password
    if [[ -n "${components[1]}" && "${components[1]}" != "<token>" && -n "${components[2]}" ]]; then
        echo >&2 "Using login for ${components[0]} @ ${registry_url} from ~/.docker/config.json"
        print_auth "$(mkauth "${components[0]}" "${components[1]}")"
    fi
}

function try_dockercfg_credstore() {
	local dockercfg="$1"
	credstore="$(jq -r <<<"$dockercfg" '.credsStore // ""')"
    [[ -n "$credstore" ]] || return
    local helper_cmd="docker-credential-${credstore}"
    if ! type "$helper_cmd" >/dev/null 2>&1 ; then
        echo >&2 "Not using keychain '${credstore}' as credentials helper is unavailable."
        return
    fi
    local creds_output
    creds_output="$("$helper_cmd" get <<<"$registry_url" 2>/dev/null)"
    [[ $? == 0 && -n "$creds_output" ]] || return
    local components=()
    IFS=$'\n' read -d '' -r -a components < <(jq -r <<<"$creds_output" '(.Username // "", .Secret // "")')
    [[ -z "$username" || "$username" == "${components[0]}" ]] || return
    # stackrox.io returns a refresh token instead of a username and password so we should fall back to
    # user input username and password
    if [[ -n "${components[0]}" && "${components[0]}" != "<token>" && -n "${components[1]}" ]]; then
        echo >&2 "Using login for ${components[0]} @ ${registry_url} from keychain '${credstore}'."
        print_auth "$(mkauth "${components[0]}" "${components[1]}")"
    fi
}


if [[ -n "$username" && -n "$password" ]]; then
	echo >&2 "Warning: providing passwords via (exported) environment variables is unsafe."
	print_auth "$(mkauth "${REGISTRY_USERNAME}" "${REGISTRY_PASSWORD}")"
fi

if [[ -f ~/.docker/config.json || ! -x "$(command -v jq)" ]]; then
	dockercfg="$(< ~/.docker/config.json)"
	try_dockercfg_plain "$dockercfg"
	try_dockercfg_credstore "$dockercfg"
fi

if [[ -z "$username" ]]; then
	read -p "Enter username for docker registry at ${registry_url}: " username
fi
[[ -n "$username" ]] || { echo >&2 "Aborted." ; exit 1 ; }
read -s -p "Enter password for ${username} @ ${registry_url}: " password
[[ -n "$password" ]] || { echo >&2 "Aborted." ; exit 1 ; }

print_auth "$(mkauth "$username" "$password")"