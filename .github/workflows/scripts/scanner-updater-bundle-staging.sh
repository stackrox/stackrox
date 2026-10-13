#!/usr/bin/env bash
# ROX-36961: one candidate per stream, with generation-safe promotion.
set -euo pipefail

sc=./.github/workflows/scripts/scanner-updater-config.sh

staging_object() { "$sc" bundle-object "${1}-staging"; }

emit_decision() {
    local action="$1" should_stage="$2" generation="${3:-}"
    printf 'action=%s\nshould_stage=%s\ngeneration=%s\n' "$action" "$should_stage" "$generation" >> "${GITHUB_OUTPUT:?}"
}

# Returns 1 only when the object is absent; all other storage errors fail.
read_staged_metadata() {
    local object="$1" error_file metadata
    error_file=$(mktemp)
    if metadata=$(gcloud storage objects describe "$object" --format=json 2>"$error_file"); then
        rm -f "$error_file"
        jq -e '.generation and .creation_time and .custom_fields["content-digest"] and .custom_fields["source-baseline"]' <<< "$metadata" >/dev/null || {
            echo "::error::Staged candidate has missing metadata: $object" >&2
            return 2
        }
        printf '%s\n' "$metadata"
    elif grep -Eqi '404|not found|No URLs matched' "$error_file"; then
        rm -f "$error_file"
        return 1
    else
        cat "$error_file" >&2
        rm -f "$error_file"
        return 2
    fi
}

# Temporary integration boundary. Replace ONLY this function with the
# authenticated Grafana query for this exact stream and whole-ZIP digest.
query_bundle_health() {
    local stream="$1" sha256="$2"
    [[ "$sha256" =~ ^[a-f0-9]{64}$ ]] || { echo "::error::Invalid SHA-256" >&2; return 1; }
    echo Normal
}

remove_staged_candidate() {
    local object="$1" generation="$2"
    # The precondition prevents deleting a replacement candidate.
    gcloud storage rm "$object" --if-generation-match="$generation"
}

evaluate_staging() {
    local stream="$1" object metadata generation digest created epoch now age health
    if [[ "${FORCE_STAGE:-false}" == true ]]; then
        echo 'Force-stage: skip evaluation; rebuild staging without publishing'
        emit_decision skip true
        return
    fi

    object=$(staging_object "$stream")
    if metadata=$(read_staged_metadata "$object"); then
        :
    else
        case "$?" in
            1) emit_decision skip true; return ;;
            *) echo '::error::Unable to inspect staging' >&2; return 1 ;;
        esac
    fi
    generation=$(jq -r '.generation' <<< "$metadata")
    digest=$(jq -r '.custom_fields["content-digest"]' <<< "$metadata")
    [[ "$generation" =~ ^[0-9]+$ && "$digest" =~ ^sha256:[a-f0-9]{64}$ ]] || {
        echo '::error::Invalid staging generation or content-digest' >&2; return 1;
    }
    [[ "$(jq -r '.custom_fields["source-baseline"]' <<< "$metadata")" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]] || { echo "::error::Invalid staged source baseline" >&2; return 1; }
    created=$(jq -r '.creation_time' <<< "$metadata")
    epoch=$(date -d "$created" +%s)
    now=$(date +%s)
    age=$((now - epoch))
    health=$(query_bundle_health "$stream" "${digest#sha256:}") || {
        echo '::error::Health query failed; keeping staged candidate' >&2; return 1;
    }

    case "$health" in
        Normal)
            emit_decision publish true "$generation"
            ;;
        Alerting)
            echo "::warning::Rejecting unhealthy staged candidate for $stream"
            remove_staged_candidate "$object" "$generation"
            emit_decision skip true
            ;;
        Pending|'No Data')
            if (( age >= 8 * 3600 )); then
                echo "::warning::Staged candidate timed out after ${age}s: $stream"
                remove_staged_candidate "$object" "$generation"
                emit_decision skip true
            else
                echo "Staged candidate awaiting validation ($health, ${age}s)"
                emit_decision skip false
            fi
            ;;
        *) echo "::error::Unexpected health state: $health" >&2; return 1 ;;
    esac
}

stage_bundle() {
    local stream="$1" zip_path="$2" object digest metadata
    [[ -s "$zip_path" ]] || { echo "::error::Missing or empty ZIP: $zip_path" >&2; return 1; }
    object=$(staging_object "$stream")
    [[ "${SOURCE_BASELINE:-}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]] || { echo "::error::Missing or invalid SOURCE_BASELINE" >&2; return 1; }
    digest=$(sha256sum "$zip_path" | awk '{print $1}')
    if [[ "${FORCE_STAGE:-false}" == true ]]; then
        # Explicit manual replacement; ordinary runs must never overwrite.
        gcloud storage cp "$zip_path" "$object" --custom-metadata="content-digest=sha256:$digest,source-baseline=$SOURCE_BASELINE"
    else
        gcloud storage cp "$zip_path" "$object" --if-generation-match=0 --custom-metadata="content-digest=sha256:$digest,source-baseline=$SOURCE_BASELINE"
    fi
    echo "Staged $stream SHA-256 $digest"
}

case "${1:-}" in
    evaluate) [[ $# -eq 2 ]] || exit 2; evaluate_staging "$2" ;;
    stage) [[ $# -eq 3 ]] || exit 2; stage_bundle "$2" "$3" ;;
    *) echo "Usage: $0 {evaluate <stream>|stage <stream> <zip>}" >&2; exit 2 ;;
esac
