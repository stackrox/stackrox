#!/usr/bin/env bash

# Historical migrators enforce the sequence floor, not the release-stream limit.
rollback_rejection_observed() {
    local pods="$1" logs="$2" image="$3" sequence="$4" minimum="$5"
    jq -e --arg image "$image" '
        (.items | length) == 1 and
        (.items[0] |
            any(.spec.containers[]; .name == "central" and .image == $image) and
            any(.status.conditions[]?; .type == "Ready" and .status == "False") and
            any(.status.containerStatuses[]?; .name == "central" and .ready == false and
                ((.state.terminated.exitCode // 0) > 0 or (.lastState.terminated.exitCode // 0) > 0)))
    ' <<<"$pods" >/dev/null || return 1
    grep -Eq "Software downgrade is not supported\..*supports database version of ${sequence} but.*at least( least)? ${minimum}([^0-9]|$)|Central rollback blocked:.*requires sequence ${minimum}, but.*only supports sequence ${sequence}\." <<<"$logs"
}

rollback_check_snapshot() {
    jq -e --argjson minimum "$2" 'length == 1 and .[0].minseqnum == $minimum' <<<"$1" >/dev/null
}

rollback_version_snapshot() {
    # Expand the password only inside the DB container, not in the host's command line.
    # shellcheck disable=SC2016
    kubectl -n stackrox exec deploy/central-db -c central-db -- sh -c '
        export PGPASSWORD="$(cat /run/secrets/stackrox.io/secrets/password)"
        exec psql -X -v ON_ERROR_STOP=1 -U postgres -d central_active -Atc \
            "SELECT coalesce(json_agg(v), '\''[]'\''::json) FROM versions v"
    ' | jq -cS .
}

stop_rollback_central() {
    kubectl -n stackrox scale deploy/central --replicas=0 || return 1
    kubectl -n stackrox wait --for=delete pod -l app=central --timeout=120s
}

set_rollback_version() {
    local tag="$1" config
    [[ "$tag" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || return 1
    config="$(kubectl -n stackrox get configmap/central-config -o json |
        jq -r '.data["central-config.yaml"]' |
        yq e ".maintenance.forceRollbackVersion = \"$tag\"" -)" || return 1
    kubectl -n stackrox patch configmap/central-config --type=merge -p \
        "$(jq -n --arg config "$config" '{data:{"central-config.yaml":$config}}')" || return 1
    # Never downgrade Central DB (or Scanner DB) along with Central.
    kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$tag" || return 1
    kubectl -n stackrox scale deploy/central --replicas=1
}

wait_for_rollback_rejection() {
    local tag="$1" sequence="$2" minimum="$3" artifacts="$4" attempts="${5:-120}"
    local attempt pods pod logs
    mkdir -p "$artifacts"
    # New migrators delay exit for five minutes; allow time to observe that exit.
    for ((attempt=0; attempt<attempts; attempt++)); do
        pods="$(kubectl -n stackrox get pods -l app=central -o json)" || return 1
        printf '%s\n' "$pods" >"$artifacts/pods.json"
        # A ready Central at any point is a failed negative test, not a retry.
        if jq -e 'any(.items[].status.conditions[]?; .type == "Ready" and .status == "True")' <<<"$pods" >/dev/null; then
            echo "Central unexpectedly became ready on N-4 ($tag)" >&2
            return 1
        fi
        pod="$(jq -r '.items[0].metadata.name // empty' <<<"$pods")"
        if [[ -n "$pod" ]]; then
            kubectl -n stackrox logs "$pod" -c central >"$artifacts/current.log" 2>&1 || true
            kubectl -n stackrox logs "$pod" -c central --previous >"$artifacts/previous.log" 2>&1 || true
            logs="$(cat "$artifacts/current.log" "$artifacts/previous.log")"
            if rollback_rejection_observed "$pods" "$logs" "$REGISTRY/main:$tag" "$sequence" "$minimum"; then
                return 0
            fi
        fi
        sleep 5
    done
    echo "No proven compatibility rejection for N-4 ($tag); see $artifacts" >&2
    return 1
}

test_rejected_rollback() {
    local plan="$1" artifacts="$2" before after minimum
    minimum="$(jq -r '.minimum_sequence' <<<"$plan")"
    stop_rollback_central || return 1
    before="$(rollback_version_snapshot)" || return 1
    rollback_check_snapshot "$before" "$minimum" || return 1
    mkdir -p "$artifacts"
    printf '%s\n' "$before" >"$artifacts/version-before.json"
    set_rollback_version "$(jq -r '.rejected.tag' <<<"$plan")" || return 1
    wait_for_rollback_rejection "$(jq -r '.rejected.tag' <<<"$plan")" \
        "$(jq -r '.rejected.sequence' <<<"$plan")" "$minimum" "$artifacts" || return 1
    stop_rollback_central || return 1
    after="$(rollback_version_snapshot)" || return 1
    printf '%s\n' "$after" >"$artifacts/version-after.json"
    if [[ "$before" != "$after" ]]; then
        echo 'Rejected rollback changed the database version record' >&2
        return 1
    fi
}

recover_rollback_central() {
    local status="$1" config="$2"
    if (( status != 0 )); then
        kubectl -n stackrox patch configmap/central-config --type=merge -p "$config" || true
        kubectl -n stackrox set image deploy/central "central=$REGISTRY/main:$CURRENT_TAG" || true
        kubectl -n stackrox scale deploy/central --replicas=1 || true
    fi
    return "$status"
}
