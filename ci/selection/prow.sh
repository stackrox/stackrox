#!/usr/bin/env bash

# ci_selection_should_skip returns success only for a verified product-test skip.
ci_selection_should_skip() {
    local target="${1:?missing test target}"
    local metadata base head decision
    command -v ci-selection >/dev/null || {
        echo 'CI selection unavailable: binary missing; retaining existing execution' >&2
        return 1
    }
    # Job metadata authorizes this gate only for an unambiguous single-PR presubmit.
    if [[ -n "${JOB_SPEC:-}" ]]; then
        metadata="$(jq -ce 'select(.type == "presubmit" and (.refs.pulls | length) == 1) | .refs' <<< "$JOB_SPEC")" || return 1
    elif [[ -n "${CLONEREFS_OPTIONS:-}" ]]; then
        metadata="$(jq -ce 'select((.refs | length) == 1 and (.refs[0].pulls | length) == 1) | .refs[0]' <<< "$CLONEREFS_OPTIONS")" || return 1
    else
        return 1
    fi
    base="$(jq -er '.base_sha | select(type == "string")' <<< "$metadata")" || return 1
    head="$(jq -er '.pulls[0].sha | select(type == "string")' <<< "$metadata")" || return 1
    if ! decision="$(ci-selection --job "$target" --base "$base" --head "$head" --ignore ci/selection/demo-ignore.json)"; then
        echo 'CI selection failed; retaining existing execution' >&2
        return 1
    fi
    printf 'CI selection: %s\n' "$decision" >&2
    if [[ -n "${ARTIFACT_DIR:-}" ]]; then
        printf '%s\n' "$decision" > "$ARTIFACT_DIR/ci-selection.json" || true
    fi
    jq -e --arg job "$target" '
      .version == 1 and .job == $job and .decision == "skip" and .reason == "docs-only" and
      .facts.version == 1 and .facts.complete == true and (.facts.every | index("docs") != null)
    ' <<< "$decision" >/dev/null
}
