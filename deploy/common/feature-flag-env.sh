#!/usr/bin/env bash

# ROX_SCANNER_V4 and ROX_LEGACY_SCANNER are not ordinary feature-flag env.
# The Central Helm chart already writes them onto the Central container from
# scannerV4 / scanner config (01-central-13-deployment.yaml.htpl). Helm
# customize.central.envVars is rendered by srox.envVars, which appends to
# that list, so injecting the same name produces a second env entry.
# Kubernetes keeps both (last wins at runtime), but kubectl apply cannot
# patch a Deployment that has duplicate env names, and tests that expect a
# single value fail. Scanner V4 / legacy scanner still follow the chart
# values; this skip only avoids copying the env name twice.
# Kubectl inject does not skip these flags: generated manifests omit those
# names, and kubectl set env --local overwrites by name.
is_helm_owned_feature_flag() {
    case "$1" in
        ROX_SCANNER_V4 | ROX_LEGACY_SCANNER) return 0 ;;
        *) return 1 ;;
    esac
}

# omit_helm_owned_feature_flags drops Helm-owned scanner flags from a stream
# of VAR=value lines. Use on the Helm inject path only.
omit_helm_owned_feature_flags() {
    local assignment var
    while IFS= read -r assignment; do
        [[ -z "${assignment}" ]] && continue
        var="${assignment%%=*}"
        if is_helm_owned_feature_flag "${var}"; then
            continue
        fi
        printf '%s\n' "${assignment}"
    done
}

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
