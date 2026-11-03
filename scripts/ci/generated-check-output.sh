#!/usr/bin/env bash
# Diagnostics for check-generated.sh. Keep checks in child shells: invoking a
# shell function in an || list would disable errexit throughout that function.

generated_check_error() {
    local file="$1" line="$2" message="$3" properties=""
    printf '%s\n' "$message" >> "$CHECK_STATE/details"
    if ! is_GITHUB_ACTIONS; then
        printf 'ERROR: %s: %s\n' "$file" "$message"
        return
    fi
    message="${message//'%'/'%25'}"
    message="${message//$'\r'/'%0D'}"
    message="${message//$'\n'/'%0A'}"
    if [[ -n "$file" ]]; then
        file="${file//'%'/'%25'}"
        file="${file//$'\r'/'%0D'}"
        file="${file//$'\n'/'%0A'}"
        file="${file//','/'%2C'}"
        file="${file//':'/'%3A'}"
        properties=" file=$file"
        [[ -z "$line" ]] || properties+=",line=$line"
    fi
    printf '::error%s::%s\n' "$properties" "$message"
}

run_generator_command() {
    : > "$CHECK_STATE/details"
    local status=0 log command
    log=$(mktemp "$CHECK_STATE/command.XXXXXX")
    printf -v command '%q ' "$@"
    github_group "$command"
    "$@" 2>&1 | tee "$log" || status=$?
    github_endgroup
    if (( status == 0 )); then return 0; fi
    generated_annotation "" "" "Command failed (exit $status): $command. Open the '$command' log group."

    # Extract selected diagnostic patterns from the output file and emit them in-line.
    # Only recognize explicit Go diagnostic locations, never infer a location
    # from free-form error text. Nested generators may change directories, so
    # relative diagnostic paths cannot safely be resolved against our cwd.
    local pattern='^(.+\.go):([0-9]+)(:([0-9]+))?: (.+)$'
    local text file line count=0
    local -A seen=()
    while IFS= read -r text; do
        [[ "$text" =~ $pattern ]] || continue
        file="${BASH_REMATCH[1]}" line="${BASH_REMATCH[2]}"
        [[ -z "${seen[$text]:-}" ]] || continue
        seen["$text"]=1
        count=$((count + 1))
        (( count <= 10 )) || continue
        if [[ "$file" == /* ]]; then
            file=$(realpath -m -- "$file")
        else
            file=""
        fi
        if [[ "$file" == "$ROOT/"* && "$line" -gt 0 ]]; then
            file="${file#"$ROOT/"}"
        else
            file="" line=""
        fi
        generated_annotation "$file" "$line" "$text — See '$command' log group."
    done < "$log"
    if (( count > 10 )); then
        generated_annotation "" "" "$((count - 10)) additional diagnostics omitted; see '$command' log group."
    fi
    return "$status"
}

generated_diff() {
    local status=0 file line count=0
    git diff --exit-code HEAD || status=$?
    (( status != 0 )) || return 0
    if (( status != 1 )); then
        generated_annotation "" "" "git diff failed (exit $status)."
        return "$status"
    fi
    while IFS= read -r -d '' file; do
        count=$((count + 1))
        (( count <= 10 )) || continue
        line=$(git diff --no-ext-diff --unified=0 HEAD -- "$file" |
            sed -nE 's/^@@ -([0-9]+)(,[0-9]+)? .*/\1/p' | sed -n '1p')
        [[ "$line" != 0 ]] || line=""
        generated_annotation "$file" "$line" "Generated output differs: $file. Regenerate the files using the instructions above and commit the result."
    done < <(git diff --name-only -z HEAD)
    if (( count > 10 )); then
        generated_annotation "" "" "$((count - 10)) additional changed files; see full diff in logs."
    fi
    return 1
}

export -f generated_annotation generated_command generated_diff
