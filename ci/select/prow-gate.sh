#!/usr/bin/env bash
# Sourced by .openshift-ci/dispatch.sh.
# ci_decision_allows returns 0 when the job should run and 1 when the
# decision skips it. A broken gate returns 0, so a script error does not
# turn a required Prow job green.

# shellcheck disable=SC2329 # dispatch.sh calls this after sourcing the file.
ci_decision_allows() {
    local job="${1:?usage: ci_decision_allows <job>}"
    local python="python3"
    if ! command -v python3 >/dev/null 2>&1; then
        info "CI decision: python3 is missing, so ${job} runs"
        return 0
    fi

    local pr_json=""
    if ! pr_json="$(get_pr_details)"; then
        info "CI decision: pull request details are unavailable, so ${job} runs"
        return 0
    fi

    local enforced=0
    if is_openshift_CI_rehearse_PR; then
        if pr_has_label_in_body "ci-dispatcher-enforce" "$pr_json"; then
            enforced=1
        fi
    elif jq -r '(.labels // [])[].name' <<<"$pr_json" | grep -qx 'ci-dispatcher-enforce'; then
        enforced=1
    fi

    local files_file labels_file comments_file
    files_file="$(mktemp)"
    labels_file="$(mktemp)"
    comments_file="$(mktemp)"

    local files_unknown=0
    local base="${PULL_BASE_SHA:-}"
    if [[ -z "$base" ]] || ! git -C "$ROOT" diff --name-only "${base}...HEAD" >"$files_file"; then
        info "CI decision: changed files are unknown"
        files_unknown=1
    fi
    if ! jq -r '(.labels // [])[].name' <<<"$pr_json" >"$labels_file"; then
        : >"$labels_file"
    fi
    # A skip is wrong when this run was started by /test and the comment
    # list could not be read. Run the job instead of hiding that command.
    if ! _ci_decision_write_comments "$comments_file" "$pr_json"; then
        info "CI decision: could not read /test comments, so ${job} runs"
        rm -f "$files_file" "$labels_file" "$comments_file"
        return 0
    fi

    local repo pr commit commit_time head_ref action
    repo="$(jq -r '.base.repo.full_name // "stackrox/stackrox"' <<<"$pr_json")"
    pr="$(jq -r '.number // 0' <<<"$pr_json")"
    # Prow checks out a synthetic merge. Its committer date can be later
    # than a /test comment on the pull-request head.
    head_ref="${PULL_PULL_SHA:-HEAD}"
    commit="$(git -C "$ROOT" rev-parse --short=7 "$head_ref" 2>/dev/null || echo unknown)"
    commit_time="$(git -C "$ROOT" show -s --format=%cI "$head_ref" 2>/dev/null || true)"

    local -a cmd
    cmd=(
        "$python" "$ROOT/ci/select/prow.py" gate
        --job "$job"
        --rules "$ROOT/ci/test-domains.toml"
        --defaults "$ROOT/ci/decision-defaults"
        --labels-file "$labels_file"
        --comments-file "$comments_file"
        --commit-time "$commit_time"
        --repo "$repo"
        --pr "$pr"
        --commit "$commit"
    )
    if [[ "$files_unknown" -eq 1 ]]; then
        cmd+=(--files-unknown)
    else
        cmd+=(--files-file "$files_file")
    fi
    if [[ "$enforced" -eq 1 ]]; then
        cmd+=(--enforce)
    fi

    action="run"
    if ! action="$("${cmd[@]}")"; then
        info "CI decision: the gate failed, so ${job} runs"
        rm -f "$files_file" "$labels_file" "$comments_file"
        return 0
    fi
    rm -f "$files_file" "$labels_file" "$comments_file"
    if [[ "$action" == "skip" ]]; then
        return 1
    fi
    return 0
}

_ci_decision_write_comments() {
    local dest="$1"
    local pr_json="$2"
    local org repo number payload
    echo '[]' >"$dest"
    org="$(jq -r '.base.repo.owner.login // empty' <<<"$pr_json")"
    repo="$(jq -r '.base.repo.name // empty' <<<"$pr_json")"
    number="$(jq -r '.number // empty' <<<"$pr_json")"
    if [[ ! "$org" =~ ^[A-Za-z0-9._-]+$ || ! "$repo" =~ ^[A-Za-z0-9._-]+$ || ! "$number" =~ ^[0-9]+$ ]]; then
        return 1
    fi
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        if ! payload="$(curl --retry 5 --retry-connrefused -fsS \
            -H "Authorization: token ${GITHUB_TOKEN}" \
            "https://api.github.com/repos/${org}/${repo}/issues/${number}/comments")"; then
            return 1
        fi
    elif ! payload="$(curl --retry 5 --retry-connrefused -fsS \
        "https://api.github.com/repos/${org}/${repo}/issues/${number}/comments")"; then
        return 1
    fi
    if ! jq -c 'if type == "array" then [.[] | {created_at, body}] else [] end' <<<"$payload" >"$dest"; then
        return 1
    fi
}
