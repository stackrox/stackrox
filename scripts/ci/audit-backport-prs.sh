#!/usr/bin/env bash

# Audit backport PRs and Jira issues for release management
# This script:
# - Fetches all backport PRs
# - Groups them by target release branch
# - Validates Jira references and metadata
# - Finds orphaned Jira issues
# - Generates a report with author mentions

set -euo pipefail

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

# Configuration
JIRA_PROJECT="ROX"
JIRA_BASE_URL="redhat.atlassian.net"
REPORT_FILE="backport-audit-report.md"

# Temp files
TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TEMP_DIR"' EXIT

PRS_FILE="$TEMP_DIR/backport-prs.json"
PROCESSED_PRS_FILE="$TEMP_DIR/processed-prs.json"
JIRA_VALIDATION_FILE="$TEMP_DIR/jira-validation.json"

# Data structures (associative arrays)
declare -A RELEASE_VERSIONS  # Maps branch to expected next version
declare -A PRS_BY_BRANCH     # Maps branch to list of PR numbers
declare -A PR_AUTHORS        # Maps PR number to author
declare -A PR_TITLES         # Maps PR number to title
declare -A PR_JIRA_KEYS      # Maps PR number to Jira keys (comma-separated)
declare -A JIRA_ISSUES       # Maps Jira key to JSON data

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Audit backport PRs and validate Jira issues for release management.

Options:
    --branches BRANCHES    Comma-separated release branches or "all" (default: all)
    --help                 Show this help message

Environment Variables (required):
    JIRA_USER              Jira username
    JIRA_TOKEN             Jira API token
    GITHUB_TOKEN           GitHub token (or use gh auth login)

Example:
    $0 --branches release-4.10,release-4.9
    $0 --branches all
EOF
}

parse_arguments() {
    RELEASE_BRANCHES="${RELEASE_BRANCHES:-all}"

    while [[ $# -gt 0 ]]; do
        case $1 in
            --branches)
                RELEASE_BRANCHES="$2"
                shift 2
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                die "Unknown option: $1. Use --help for usage information."
                ;;
        esac
    done
}

validate_environment() {
    info "Validating environment"
    require_environment "JIRA_USER"
    require_environment "JIRA_TOKEN"

    if ! command -v gh >/dev/null 2>&1; then
        die "gh CLI is required but not installed"
    fi

    if ! command -v jq >/dev/null 2>&1; then
        die "jq is required but not installed"
    fi
}

detect_release_versions() {
    info "Detecting release versions"

    local branches
    if [[ "$RELEASE_BRANCHES" == "all" ]]; then
        # Auto-detect all release branches
        branches=$(git branch -r | grep -oE 'origin/release-[0-9]+\.[0-9]+' | sed 's|origin/||' | sort -u)
    else
        # Use provided branches
        branches=$(echo "$RELEASE_BRANCHES" | tr ',' '\n')
    fi

    while IFS= read -r branch; do
        [[ -z "$branch" ]] && continue

        # Extract version from branch name (e.g., release-4.6 → 4.6)
        if [[ "$branch" =~ ^release-([0-9]+\.[0-9]+)$ ]]; then
            local base_version="${BASH_REMATCH[1]}"

            # Find latest tag for this version
            local latest_tag
            latest_tag=$(git tag | grep -E "^${base_version}\.[0-9]+$" | sort -V | tail -1 || echo "")

            if [[ -n "$latest_tag" ]]; then
                # Extract patch version and increment
                if [[ "$latest_tag" =~ ^${base_version}\.([0-9]+)$ ]]; then
                    local patch="${BASH_REMATCH[1]}"
                    local next_patch=$((patch + 1))
                    local expected_version="${base_version}.${next_patch}"
                else
                    warn "Could not parse tag: $latest_tag"
                    continue
                fi
            else
                # No tags yet, assume .0 will be first release
                local expected_version="${base_version}.0"
            fi

            RELEASE_VERSIONS["$branch"]="$expected_version"
            info "  $branch → $expected_version (latest tag: ${latest_tag:-none})"
        else
            warn "Branch $branch does not match expected format release-X.Y"
        fi
    done <<< "$branches"

    if [[ ${#RELEASE_VERSIONS[@]} -eq 0 ]]; then
        die "No valid release branches detected. Use --branches to specify."
    fi
}

fetch_backport_prs() {
    info "Fetching backport PRs from GitHub"

    # Fetch OPEN PRs with backport label targeting release branches
    gh pr list \
        --repo stackrox/stackrox \
        --search "label:backport" \
        --state open \
        --limit 1000 \
        --json number,title,author,baseRefName,body,state \
        > "$PRS_FILE"

    local pr_count
    pr_count=$(jq 'length' "$PRS_FILE")
    info "Found $pr_count total PRs with backport label"

    # Filter and group by release branch
    for branch in "${!RELEASE_VERSIONS[@]}"; do
        local prs
        prs=$(jq -r --arg branch "$branch" \
            '[.[] | select(.baseRefName == $branch) | .number] | join(",")' \
            "$PRS_FILE")

        if [[ -n "$prs" && "$prs" != "null" ]]; then
            PRS_BY_BRANCH["$branch"]="$prs"
            local count
            count=$(echo "$prs" | tr ',' '\n' | wc -l)
            info "  $branch: $count PRs"
        fi
    done
}

resolve_authors() {
    info "Resolving PR authors (handling rhacs-bot)"

    while IFS= read -r pr_json; do
        local pr_number
        local author
        local title
        local body

        pr_number=$(echo "$pr_json" | jq -r '.number')
        author=$(echo "$pr_json" | jq -r '.author.login')
        title=$(echo "$pr_json" | jq -r '.title')
        body=$(echo "$pr_json" | jq -r '.body // ""')

        # Check if author is rhacs-bot
        if [[ "$author" == "rhacs-bot" ]]; then
            # Try to extract original PR number from body
            # Pattern: "Backport <commit> from #12345"
            if [[ "$body" =~ from\ \#([0-9]+) ]]; then
                local original_pr="${BASH_REMATCH[1]}"
                local real_author
                real_author=$(gh pr view "$original_pr" --json author --jq '.author.login' 2>/dev/null || echo "rhacs-bot")

                # If original author is dependabot, find who added the backport label
                if [[ "$real_author" == "app/dependabot" ]]; then
                    local label_adder
                    label_adder=$(gh api "repos/stackrox/stackrox/issues/${original_pr}/events" \
                        --jq '.[] | select(.event == "labeled" and (.label.name | contains("backport"))) | .actor.login' \
                        2>/dev/null | head -1 || echo "")

                    if [[ -n "$label_adder" && "$label_adder" != "github-actions[bot]" ]]; then
                        author="$label_adder"
                        info "  PR #$pr_number: rhacs-bot → @app/dependabot → @$label_adder (from #$original_pr, added backport label)"
                    else
                        author="$real_author"
                        info "  PR #$pr_number: rhacs-bot → @$real_author (from #$original_pr)"
                    fi
                else
                    author="$real_author"
                    info "  PR #$pr_number: rhacs-bot → @$real_author (from #$original_pr)"
                fi
            else
                warn "  PR #$pr_number: Could not extract original PR from rhacs-bot backport"
            fi
        fi

        # Check if author is dependabot directly (not via rhacs-bot) - find who added the backport label
        if [[ "$author" == "app/dependabot" ]]; then
            local label_adder
            label_adder=$(gh api "repos/stackrox/stackrox/issues/${pr_number}/events" \
                --jq '.[] | select(.event == "labeled" and (.label.name | contains("backport"))) | .actor.login' \
                2>/dev/null | head -1 || echo "")

            if [[ -n "$label_adder" && "$label_adder" != "github-actions[bot]" ]]; then
                author="$label_adder"
                info "  PR #$pr_number: app/dependabot → @$label_adder (added backport label)"
            fi
        fi

        PR_AUTHORS["$pr_number"]="$author"
        PR_TITLES["$pr_number"]="$title"
    done < <(jq -c '.[]' "$PRS_FILE")
}

extract_jira_keys() {
    info "Extracting Jira keys from PR titles"

    local prs_without_jira=0

    for pr_number in "${!PR_AUTHORS[@]}"; do
        local title="${PR_TITLES[$pr_number]}"
        local jira_keys

        # Extract ROX-<number> pattern (grep returns 1 if no match, so use || true)
        jira_keys=$(echo "$title" | grep -oE 'ROX-[0-9]+' | sort -u | tr '\n' ',' | sed 's/,$//' || true)

        if [[ -n "$jira_keys" ]]; then
            PR_JIRA_KEYS["$pr_number"]="$jira_keys"
        else
            PR_JIRA_KEYS["$pr_number"]=""
            ((prs_without_jira++)) || true
        fi
    done

    info "  PRs without Jira reference: $prs_without_jira"
}

get_jira_issue() {
    local issue_key="$1"
    local response

    response=$(curl --fail -sSL \
        -u "${JIRA_USER}:${JIRA_TOKEN}" \
        "https://${JIRA_BASE_URL}/rest/api/3/issue/${issue_key}?fields=fixVersions,versions,summary,status,assignee,components,customfield_10001" \
        2>/dev/null || echo "{}")

    echo "$response"
}

validate_jira_issues() {
    info "Validating Jira issues"

    # Collect all unique Jira keys
    local all_jira_keys=()
    for pr_number in "${!PR_JIRA_KEYS[@]}"; do
        local keys="${PR_JIRA_KEYS[$pr_number]}"
        if [[ -n "$keys" ]]; then
            IFS=',' read -ra key_array <<< "$keys"
            for key in "${key_array[@]}"; do
                all_jira_keys+=("$key")
            done
        fi
    done

    # Get unique keys
    local unique_keys
    unique_keys=$(printf '%s\n' "${all_jira_keys[@]}" | sort -u)

    # Validate each Jira issue
    while IFS= read -r jira_key; do
        [[ -z "$jira_key" ]] && continue

        local issue_data
        issue_data=$(get_jira_issue "$jira_key")

        if [[ "$(echo "$issue_data" | jq -r '.key // empty')" == "$jira_key" ]]; then
            JIRA_ISSUES["$jira_key"]="$issue_data"
        else
            warn "  Could not fetch Jira issue: $jira_key"
        fi
    done <<< "$unique_keys"

    info "  Validated ${#JIRA_ISSUES[@]} Jira issues"
}

find_orphaned_issues() {
    info "Finding orphaned Jira issues (issues without PRs)"

    # Initialize orphaned issues file
    echo "[]" > "$TEMP_DIR/orphaned-issues.json"

    for branch in "${!RELEASE_VERSIONS[@]}"; do
        local expected_version="${RELEASE_VERSIONS[$branch]}"

        # Query Jira for issues with this fixVersion
        local jql="project = ${JIRA_PROJECT} AND fixVersion = \"${expected_version}\""
        local response

        response=$(curl --fail -sSL \
            -u "${JIRA_USER}:${JIRA_TOKEN}" \
            --get \
            --data-urlencode "jql=${jql}" \
            --data-urlencode "fields=key,summary" \
            --data-urlencode "maxResults=1000" \
            "https://${JIRA_BASE_URL}/rest/api/3/search" \
            2>/dev/null || echo '{"issues":[]}')

        local jira_issue_keys
        jira_issue_keys=$(echo "$response" | jq -r '.issues[].key')

        # Get Jira keys from PRs targeting this branch
        local pr_jira_keys=()
        if [[ -n "${PRS_BY_BRANCH[$branch]:-}" ]]; then
            IFS=',' read -ra pr_numbers <<< "${PRS_BY_BRANCH[$branch]}"
            for pr_number in "${pr_numbers[@]}"; do
                local keys="${PR_JIRA_KEYS[$pr_number]:-}"
                if [[ -n "$keys" ]]; then
                    IFS=',' read -ra key_array <<< "$keys"
                    pr_jira_keys+=("${key_array[@]}")
                fi
            done
        fi

        # Find orphaned issues (in Jira but not in any PR)
        while IFS= read -r jira_key; do
            [[ -z "$jira_key" ]] && continue

            local found=false
            for pr_key in "${pr_jira_keys[@]}"; do
                if [[ "$pr_key" == "$jira_key" ]]; then
                    found=true
                    break
                fi
            done

            if [[ "$found" == "false" ]]; then
                local summary
                summary=$(echo "$response" | jq -r --arg key "$jira_key" \
                    '.issues[] | select(.key == $key) | .summary')

                echo "$response" | jq --arg key "$jira_key" --arg branch "$branch" \
                    '.issues[] | select(.key == $key) | . + {branch: $branch}' \
                    >> "$TEMP_DIR/orphaned-issues.json.tmp"
            fi
        done <<< "$jira_issue_keys"
    done

    # Consolidate orphaned issues
    if [[ -f "$TEMP_DIR/orphaned-issues.json.tmp" ]]; then
        jq -s '.' "$TEMP_DIR/orphaned-issues.json.tmp" > "$TEMP_DIR/orphaned-issues.json"
    fi
}

generate_report() {
    info "Generating report"

    {
        echo "# Backport PR Audit Report"
        echo ""
        echo "Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo ""

        for branch in $(printf '%s\n' "${!RELEASE_VERSIONS[@]}" | sort -V); do
            local expected_version="${RELEASE_VERSIONS[$branch]}"

            # Check if this branch has anything to report
            local has_content=false

            # Check for PRs
            if [[ -n "${PRS_BY_BRANCH[$branch]:-}" ]]; then
                has_content=true
            fi

            # Check for orphaned issues
            local orphaned_check
            orphaned_check=$(jq -r --arg branch "$branch" \
                '[.[] | select(.branch == $branch) | .key] | join("\n")' \
                "$TEMP_DIR/orphaned-issues.json" 2>/dev/null || echo "")
            if [[ -n "$orphaned_check" ]]; then
                has_content=true
            fi

            # Skip empty releases
            if [[ "$has_content" == "false" ]]; then
                continue
            fi

            echo "## $branch (Expected: $expected_version)"
            echo ""

            # PRs without Jira reference
            local prs_no_jira=()
            if [[ -n "${PRS_BY_BRANCH[$branch]:-}" ]]; then
                IFS=',' read -ra pr_numbers <<< "${PRS_BY_BRANCH[$branch]}"
                for pr_number in "${pr_numbers[@]}"; do
                    if [[ -z "${PR_JIRA_KEYS[$pr_number]}" ]]; then
                        prs_no_jira+=("$pr_number")
                    fi
                done
            fi

            if [[ ${#prs_no_jira[@]} -gt 0 ]]; then
                echo "### PRs Missing Jira Reference (${#prs_no_jira[@]})"
                echo ""

                # Collect PR info for sorting
                local pr_lines=()
                for pr_number in "${prs_no_jira[@]}"; do
                    local author="${PR_AUTHORS[$pr_number]}"
                    local title="${PR_TITLES[$pr_number]}"
                    local slack_id
                    slack_id=$("$SCRIPTS_ROOT/scripts/ci/get-slack-user-id.sh" "$author" 2>/dev/null || echo "")

                    if [[ -n "$slack_id" ]]; then
                        pr_lines+=("$author|$slack_id|$pr_number|$title")
                    else
                        pr_lines+=("$author||$pr_number|$title")
                    fi
                done

                # Sort by author and output
                printf '%s\n' "${pr_lines[@]}" | sort -t'|' -k1,1 | while IFS='|' read -r author slack_id pr_number title; do
                    local mention
                    if [[ "$author" == "app/red-hat-konflux" ]]; then
                        mention=":konflux:"
                    elif [[ -n "$slack_id" ]]; then
                        mention="<@$slack_id>"
                    else
                        mention="@$author"
                    fi
                    echo "- $mention <https://github.com/stackrox/stackrox/pull/$pr_number|#$pr_number>: $title"
                done
                echo ""
            fi

            # Jira issues with missing metadata
            local issues_with_problems=()
            declare -A jira_to_prs  # Map jira_key to PR numbers

            if [[ -n "${PRS_BY_BRANCH[$branch]:-}" ]]; then
                IFS=',' read -ra pr_numbers <<< "${PRS_BY_BRANCH[$branch]}"
                for pr_number in "${pr_numbers[@]}"; do
                    local keys="${PR_JIRA_KEYS[$pr_number]:-}"
                    if [[ -n "$keys" ]]; then
                        IFS=',' read -ra key_array <<< "$keys"
                        for jira_key in "${key_array[@]}"; do
                            if [[ -n "${JIRA_ISSUES[$jira_key]:-}" ]]; then
                                local issue_data="${JIRA_ISSUES[$jira_key]}"
                                local fix_versions
                                local affected_versions
                                local assignee
                                local team
                                local component

                                fix_versions=$(echo "$issue_data" | jq -r '[.fields.fixVersions[].name] | join(", ")')
                                affected_versions=$(echo "$issue_data" | jq -r '[.fields.versions[].name] | join(", ")')
                                assignee=$(echo "$issue_data" | jq -r '.fields.assignee.displayName // "Unassigned"')
                                team=$(echo "$issue_data" | jq -r '.fields.customfield_10001.name // "No team"')
                                component=$(echo "$issue_data" | jq -r '[.fields.components[].name] | join(", ") | if . == "" then "No component" else . end')

                                local has_fix_version=":white_check_mark:"
                                local has_affected_version=":white_check_mark:"

                                if [[ -z "$fix_versions" ]] || [[ "$fix_versions" == "null" ]] || ! echo "$fix_versions" | grep -q "$expected_version"; then
                                    has_fix_version=":x:"
                                fi

                                if [[ -z "$affected_versions" ]] || [[ "$affected_versions" == "null" ]]; then
                                    has_affected_version=":x:"
                                fi

                                if [[ "$has_fix_version" == ":x:" ]] || [[ "$has_affected_version" == ":x:" ]]; then
                                    issues_with_problems+=("$jira_key|$has_fix_version|$has_affected_version|$assignee|$team|$component")

                                    # Track PR references
                                    if [[ -n "${jira_to_prs[$jira_key]:-}" ]]; then
                                        jira_to_prs[$jira_key]="${jira_to_prs[$jira_key]},$pr_number"
                                    else
                                        jira_to_prs[$jira_key]="$pr_number"
                                    fi
                                fi
                            fi
                        done
                    fi
                done
            fi

            # Remove duplicates
            if [[ ${#issues_with_problems[@]} -gt 0 ]]; then
                local unique_issues
                unique_issues=$(printf '%s\n' "${issues_with_problems[@]}" | sort -u)

                echo "### Jira Issues with Missing Metadata ($(echo "$unique_issues" | wc -l))"
                echo ""
                while IFS= read -r issue_line; do
                    IFS='|' read -r jira_key has_fix has_affected assignee team component <<< "$issue_line"

                    # Get PRs that reference this Jira issue
                    local pr_refs="${jira_to_prs[$jira_key]:-}"
                    local pr_links=""
                    if [[ -n "$pr_refs" ]]; then
                        IFS=',' read -ra pr_array <<< "$pr_refs"
                        local pr_link_list=()
                        for pr in "${pr_array[@]}"; do
                            pr_link_list+=("<https://github.com/stackrox/stackrox/pull/$pr|#$pr>")
                        done
                        pr_links=" (PRs: $(IFS=', '; echo "${pr_link_list[*]}"))"
                    fi

                    echo "- <https://${JIRA_BASE_URL}/browse/$jira_key|$jira_key>: $has_fix fixVersion, $has_affected affectedVersion (Assignee: $assignee, Team: $team, Component: $component)$pr_links"
                done <<< "$unique_issues"
                echo ""
            fi

            # Orphaned Jira issues
            local orphaned
            orphaned=$(jq -r --arg branch "$branch" \
                '[.[] | select(.branch == $branch) | .key] | join("\n")' \
                "$TEMP_DIR/orphaned-issues.json" 2>/dev/null || echo "")

            if [[ -n "$orphaned" ]]; then
                local count
                count=$(echo "$orphaned" | wc -l)
                echo "### Orphaned Jira Issues ($count)"
                echo ""
                echo "Issues with fixVersion=$expected_version but no corresponding PR:"
                echo ""
                while IFS= read -r jira_key; do
                    [[ -z "$jira_key" ]] && continue
                    echo "- <https://${JIRA_BASE_URL}/browse/$jira_key|$jira_key>"
                done <<< "$orphaned"
                echo ""
            fi
        done
    } > "$REPORT_FILE"

    info "Report written to $REPORT_FILE"
}

main() {
    parse_arguments "$@"
    validate_environment
    detect_release_versions
    fetch_backport_prs
    resolve_authors
    extract_jira_keys
    validate_jira_issues
    find_orphaned_issues
    generate_report

    info "✅ Audit complete"
}

main "$@"
