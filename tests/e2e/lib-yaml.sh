#!/usr/bin/env bash
# shellcheck disable=SC1091

set -euo pipefail

# For roxie-based deployments we use yq to manipulate the YAML overrides file, and these functions are high-level helpers for that.
# The functions modify the file in-place.

merge_yaml() {
    local input="$1"
    local tmpfile; tmpfile="$(mktemp)"

    yq eval-all '(select(fi == 0) // {}) * select(fi == 1)' "$input" <(cat) > "$tmpfile"
    cat "$tmpfile" > "$input"
    rm -f "$tmpfile"
}

patch_yaml() {
    local input="$1"
    local patch="$2"

    yq -i eval "$patch" "$input"
}

set_custom_env() {
    local override_file="$1"
    local component="$2"
    local name="$3"
    local value="$4"

    local path=".${component}.spec.customize.envVars"

    # Needs to be initialized to list first, if doesn't exist.
    init_yaml_path_as_list "$override_file" "$path"
    NAME="$name" VALUE="$value" patch_yaml "$override_file" "${path} += {\"name\": strenv(NAME), \"value\": strenv(VALUE)}"
}

set_overlay_env() {
    local override_file="$1"
    local component="$2"
    local api_version="$3"
    local kind="$4"
    local resource_name="$5"
    local container="$6"
    local name="$7"
    local value="$8"

    local overlays=".${component}.spec.overlays"

    # Initialize the overlays list if it doesn't exist.
    init_yaml_path_as_list "$override_file" "$overlays"
    # Add a new overlay element for the given api_version/kind/name if it doesn't already exist.
    local new_empty_overlay="{\"apiVersion\": \"${api_version}\", \"kind\": \"${kind}\", \"name\": \"${resource_name}\", \"patches\": []}"
    patch_yaml "$override_file" "${overlays} += [${new_empty_overlay}] | ${overlays} |= unique_by([.name, .kind])"

    # Add new patch.
    local env_path="spec.template.spec.containers[name:${container}].env"
    local env_entry="{\"name\": strenv(NAME), \"value\": strenv(VALUE)}"
    local patch="{\"path\":\"${env_path}[-1]\", \"value\": (${env_entry} | toyaml)}"
    NAME="$name" VALUE="$value" patch_yaml "$override_file" "(${overlays}[] | select(.name == \"${resource_name}\" and .kind == \"${kind}\").patches) += $patch"
}

init_yaml_path_as_list() {
    local override_file="$1"
    local path="$2"

    patch_yaml "$override_file" "${path} = (${path} // [])"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        echo >&2 "Error: When invoked at the command line a method is required."
        exit 1
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
