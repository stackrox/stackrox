#!/usr/bin/env bash
# Query source-config.yaml for scanner updater workflows.
#
# Usage:
#   scanner-updater-config.sh <command> [args...]
#
# Commands:
#   bucket                      Storage bucket name.
#   prefix                      Storage prefix path.
#   sources <stream>            List updater sources for a stream, one per line.
#   source-object <stream> <source>  GCS object path for a source.
#   bundle-object <stream>      GCS object path for a bundle.
#   interval-secs <source>      Updater refresh interval in seconds.
#   attempt-cadence-secs <source>  Attempt cadence in seconds; 0 if unset.
#
# Inputs (environment):
#   SOURCE_CONFIG — path to source-config.yaml
#
# Requires: yq.

set -euo pipefail

config="${SOURCE_CONFIG:?SOURCE_CONFIG is required}"

if [[ ! -f "$config" ]]; then
    echo "::error::config file not found: $config" >&2
    exit 1
fi

if ! yq eval '.' "$config" > /dev/null 2>&1; then
    echo "::error::config file is not valid YAML: $config" >&2
    exit 1
fi

duration_to_seconds() {
    local dur="${1:?missing required argument 'duration'}"; shift
    local total=0 remaining="$dur"
    while [[ -n "$remaining" ]]; do
        if [[ "$remaining" =~ ^([0-9]+)h(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] * 3600 )); remaining="${BASH_REMATCH[2]}"
        elif [[ "$remaining" =~ ^([0-9]+)m(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] * 60 )); remaining="${BASH_REMATCH[2]}"
        elif [[ "$remaining" =~ ^([0-9]+)s(.*)$ ]]; then
            total=$(( total + BASH_REMATCH[1] )); remaining="${BASH_REMATCH[2]}"
        else
            echo "::error::cannot parse duration: $remaining" >&2; exit 1
        fi
    done
    echo "$total"
}

command="${1:?missing required argument 'command'}"; shift

case "$command" in
    bucket)
        yq eval '.storage.bucket' "$config"
        ;;
    prefix)
        yq eval '.storage.prefix' "$config"
        ;;
    sources)
        stream="${1:?missing required argument 'stream'}"; shift
        yq eval ".bundles.\"${stream}\"[]" "$config"
        ;;
    source-object)
        stream="${1:?missing required argument 'stream'}"; shift
        source="${1:?missing required argument 'source'}"; shift
        bucket=$(yq eval '.storage.bucket' "$config")
        prefix=$(yq eval '.storage.prefix' "$config")
        echo "gs://${bucket}/${prefix}/${stream}/sources/${source}.json.zst"
        ;;
    bundle-object)
        stream="${1:?missing required argument 'stream'}"; shift
        bucket=$(yq eval '.storage.bucket' "$config")
        prefix=$(yq eval '.storage.prefix' "$config")
        echo "gs://${bucket}/${prefix}/${stream}/vulnerabilities.zip"
        ;;
    interval-secs)
        source="${1:?missing required argument 'source'}"; shift
        default_interval=$(yq eval '.updaters.default.interval // "4h"' "$config")
        interval=$(yq eval ".updaters.\"${source}\".interval // \"$default_interval\"" "$config")
        duration_to_seconds "$interval"
        ;;
    attempt-cadence-secs)
        source="${1:?missing required argument 'source'}"; shift
        cadence=$(yq eval ".updaters.\"${source}\".attempt_cadence // .updaters.default.attempt_cadence // \"\"" "$config")
        if [[ -z "$cadence" ]]; then
            echo 0
            exit 0
        fi
        cadence_secs=$(duration_to_seconds "$cadence")
        if (( cadence_secs <= 0 || cadence_secs % 3600 != 0 )); then
            echo "::error::attempt_cadence must be a positive whole-hour duration: $cadence" >&2
            exit 1
        fi
        echo "$cadence_secs"
        ;;
    *)
        echo "::error::unknown command: $command" >&2
        exit 1
        ;;
esac
