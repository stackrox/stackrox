#!/usr/bin/env bash

# The output appears in a dedicated emphasized GitHub step box.
# Multiple lines are not supported.
# Markdown is not supported.
gh_log() {
    local LEVEL="$1"
    shift
    if [ "$CI" = "false" ]; then
        case "$LEVEL" in
        "debug")
            printf "\e[1;90m"
            ;;
        "error")
            printf "\e[1;31m"
            ;;
        "warning")
            printf "\e[1;33m"
            ;;
        *)
            printf "\e[1;32m"
            ;;
        esac
    fi
    echo "::$LEVEL::$*"
    if [ "$CI" = "false" ]; then printf "\e[0m"; fi
}
export -f gh_log

# Sets a step output value.
# Examples:
#     gh_output name value
#     gh_output name <<EOF
#     value
#     EOF
gh_output() {
    local NAME="$1"
    if [ "$#" -eq 1 ]; then
        echo "$NAME<<END_OF_VALUE"
        cat
        echo "END_OF_VALUE"
    else
        shift
        echo "$NAME=$*"
    fi >>"$GITHUB_OUTPUT"
}
export -f gh_output

check_not_empty() {
    for V in "$@"; do
        typeset -n VAR="$V"
        if [ -z "${VAR:-}" ]; then
            gh_log error "Variable $V is not set or empty"
            exit 1
        fi
    done
}
export -f check_not_empty

check_not_empty GITHUB_STEP_SUMMARY

# The output appears in the GitHub step summary box.
# Multiple lines are supported.
# Markdown is supported.
# Examples:
#     gh_summary "Markdown summary"
#     gh_summary <<EOF
#     Markdown summary
#     EOF
gh_summary() {
    if [ "$CI" = "false" ]; then printf "\e[94m"; fi
    if [ "$#" -eq 0 ]; then
        cat # for the data passed via the pipe
    else
        echo -e "$@" # for the data passed as arguments
    fi >>"$GITHUB_STEP_SUMMARY"
    if [ "$CI" = "false" ]; then printf "\e[0m"; fi
}
export -f gh_summary

##########
## JIRA ##
##########
jira_check_not_empty() {
    check_not_empty \
        JIRA_USER \
        JIRA_TOKEN \
        JIRA_BASE_URL
}
export -f jira_check_not_empty

# Retrieves issues from JIRA using JQL and stores them in JSON format in the OUTPUT_FILE.
# If additional fields are needed, they can be added to the "fields" array.
#
# Example:
#   get_issues "project = ROX AND fixVersion = 4.9.0" "issues.json"
get_issues() {
    jira_check_not_empty
    local JQL="$1"
    local OUTPUT_FILE="$2"

    # Initializations
    PAGE_SIZE=20
    next_page_token=""
    echo "[]" > "${OUTPUT_FILE}"

    # Paginate through the results
    while true; do
        cat <<EOF > /tmp/request.json
{
    "jql": "$JQL",
    "maxResults": "${PAGE_SIZE}",
    "nextPageToken": "${next_page_token}",
    "fields": [
        "key",
        "assignee",
        "issuetype",
        "status",
        "summary"
    ]
}
EOF

        response=$(curl --fail -sSL --request POST \
            --url "https://${JIRA_BASE_URL}/rest/api/3/search/jql" \
            --user "${JIRA_USER}:${JIRA_TOKEN}" \
            --header "Accept: application/json" \
            --header "Content-Type: application/json" \
            --data-binary @/tmp/request.json \
            --retry 5 --retry-all-errors
        )
        if [ -z "${response}" ]; then
            gh_log error "Failed to retrieve issues from JIRA."
            exit 1
        fi

        # Append the issues from the current page to $OUTPUT_FILE
        # Need to use a temporary file as jq does not support in-place editing.
        jq --argjson new_issues "$(echo "$response" | jq '.issues')" '. + $new_issues' "${OUTPUT_FILE}" > "${OUTPUT_FILE}.tmp"
        mv "${OUTPUT_FILE}.tmp" "${OUTPUT_FILE}"

        if [[ "$(echo "${response}" | jq '.isLast')" = "true" ]]; then
            # No more pages, exit the loop.
            break
        else
            next_page_token="$(echo "${response}" | jq -r '.nextPageToken')"
        fi
    done
}
export -f get_issues

# Retrieves a JIRA issue with ISSUE_KEY and returns it in JSON format.
#
# Example:
#   get_issue "ROX-29997" > rox-29997.json
get_issue() {
    jira_check_not_empty
    local ISSUE_KEY="$1"

    curl --fail -sSL --request GET \
    --user "${JIRA_USER}:${JIRA_TOKEN}" \
    --header "Accept: application/json" \
    --retry 5 --retry-all-errors \
    "https://${JIRA_BASE_URL}/rest/api/3/issue/${ISSUE_KEY}"
}
export -f get_issue

# Adds COMMENT to a JIRA issue with ISSUE_KEY.
#
# Example:
#   comment_on_issue "ROX-29997" "Please update the status of this issue and notify the release engineer."
comment_on_issue() {
    jira_check_not_empty
    local ISSUE_KEY="$1"
    local COMMENT="$2"

    ## Commenting on a single issue
    cat <<EOF > /tmp/request.json
{
    "body": {
        "version": 1,
        "type": "doc",
        "content": [
            {
                "type": "paragraph",
                "content": [
                    {
                    "type": "text",
                    "text": "${COMMENT}"
                    }
                ]
            }
        ]
    }
}
EOF

    curl --fail --output /dev/null -sSL --request POST \
    --user "${JIRA_USER}:${JIRA_TOKEN}" \
    --header "Accept: application/json" \
    --header "Content-Type: application/json" \
    --data-binary @/tmp/request.json \
    "https://${JIRA_BASE_URL}/rest/api/3/issue/${ISSUE_KEY}/comment"

    gh_log debug "Commented on issue ${ISSUE_KEY}"
}
export -f comment_on_issue

get_jira_release() {
    jira_check_not_empty
    local PROJECT="$1"
    local VERSION="$2"

    curl --fail -sSL --request GET \
    --user "${JIRA_USER}:${JIRA_TOKEN}" \
    --header "Accept: application/json" \
    --retry 5 --retry-all-errors \
    "https://${JIRA_BASE_URL}/rest/api/3/project/${PROJECT}/versions" | jq -r ".[] | select(.name == \"${VERSION}\")"
}
export -f get_jira_release

# Fetches the CHANGELOG from a release branch
# and returns the content of the section for a given version.
fetch_changelog() {
    RELEASE_BRANCH="$1"
    VERSION="$2"

    check_not_empty \
      RELEASE_BRANCH \
      VERSION

    ESCAPED_VERSION="${VERSION//./\.}"
    CHANGELOG="$(gh api \
      -H "Accept: application/vnd.github.v3.raw" \
      "/repos/${GITHUB_REPOSITORY}/contents/CHANGELOG.md?ref=${RELEASE_BRANCH}"
    )"

    echo "$CHANGELOG" | sed -n "/^## \[$ESCAPED_VERSION]$/,/^## \[/p" | sed '1d;$d' | sed '/./,$!d'
}
export -f fetch_changelog

# bash trick to check if the script is sourced.
if ! (return 0 2>/dev/null); then # called
    SCRIPT="$1"
    check_not_empty \
        SCRIPT \
        GITHUB_REPOSITORY \
        GITHUB_REF_NAME

    URL="/repos/$GITHUB_REPOSITORY/contents/.github/workflows/scripts/$SCRIPT.sh?ref=$GITHUB_REF_NAME"
    shift
    gh_log debug "Executing '$SCRIPT.sh' from '$GITHUB_REPOSITORY' $GITHUB_REF_NAME branch with: ${*@Q}"
    gh api -H "Accept: application/vnd.github.v3.raw" "$URL" | bash -s -- "$@"
fi
