#!/usr/bin/env bash

# feature_flag_env_assignments prints VAR=value for every pkg/features flag
# that is set and non-empty in this process. Release roxctl generate does not
# copy these into the bundle; callers apply them to Central after generate.
feature_flag_env_assignments() {
    local list_file="$1"
    local var

    if [[ ! -f "${list_file}" ]]; then
        echo >&2 "WARNING: feature flag list ${list_file} not found; Central will not receive feature-flag env"
        return 0
    fi

    while IFS= read -r var; do
        if [[ -n "${!var:-}" ]]; then
            printf '%s=%s\n' "${var}" "${!var}"
        fi
    done < <(grep -oE '"ROX_[A-Z0-9_]+"' "${list_file}" | tr -d '"' | sort -u)
}
