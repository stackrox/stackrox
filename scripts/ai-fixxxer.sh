#!/bin/bash
set -euo pipefail

: "${PR_NUMBER:?PR_NUMBER is required}"
: "${REPO:?REPO is required}"
: "${GH_TOKEN:?GH_TOKEN is required}"

MANIFEST="/tmp/ai-fixxxer-results.json"

post_reply() {
    local cid="$1"
    local body="$2"
    jq -n --arg b "$body" '{"body": $b}' > /tmp/reply_payload.json
    local http_code
    http_code=$(curl -s -o /tmp/reply_response.json -w "%{http_code}" \
        -X POST "https://api.github.com/repos/${REPO}/pulls/${PR_NUMBER}/comments/${cid}/replies" \
        -H "Authorization: Bearer ${GH_TOKEN}" \
        -H "Accept: application/vnd.github+json" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        -H "Content-Type: application/json" \
        -d @/tmp/reply_payload.json)
    if [ "${http_code}" = "201" ]; then
        echo "Replied to comment ${cid}"
    else
        echo "Failed to reply to comment ${cid} (HTTP ${http_code})"
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Single-comment mode: address one specific review comment
# ──────────────────────────────────────────────────────────────────────
if [ -n "${SINGLE_COMMENT_ID:-}" ]; then
    echo "AI Fixxxer: Single-comment mode triggered for comment ${SINGLE_COMMENT_ID} on PR #${PR_NUMBER}"

    # --- Debounce: wait for more /ai-fixxx replies to accumulate ---
    echo "Debouncing for 30 seconds..."
    sleep 30

    # Fetch ALL comments on the PR (top-level and replies)
    echo "Fetching all PR comments..."
    all_comments=$(gh api "repos/${REPO}/pulls/${PR_NUMBER}/comments" --paginate)

    # Find /ai-fixxx reply comments and their parent IDs
    # Then check which parent comments already have an AI Fixxxer response
    # (a sibling reply starting with "Fixed", "**Skipped**", "**Not addressed**", or "**AI Fixxxer**")
    pending=$(echo "$all_comments" | jq '
      # All /ai-fixxx reply comments (have in_reply_to_id set)
      [.[] | select(.in_reply_to_id != null and (.body | startswith("/ai-fixxx")))] as $requests |
      # Parent IDs that already have a bot response
      [.[] | select(
        .in_reply_to_id != null and (
          (.body | startswith("Fixed ")) or
          (.body | startswith("**Skipped**")) or
          (.body | startswith("**Not addressed**")) or
          (.body | startswith("**AI Fixxxer**"))
        )
      ) | .in_reply_to_id] as $addressed |
      # Keep only requests whose parent is NOT yet addressed
      [$requests[] | select(.in_reply_to_id as $pid | $addressed | index($pid) | not)]
    ')

    pending_count=$(echo "$pending" | jq 'length')
    echo "Found ${pending_count} pending /ai-fixxx request(s)"

    if [ "$pending_count" -eq 0 ]; then
        echo "All /ai-fixxx requests on this PR already addressed. Skipping."
        exit 0
    fi

    # Build array of parent comments to process, with per-comment user instructions
    parent_ids=$(echo "$pending" | jq '[.[].in_reply_to_id] | unique')
    structured_comments=$(echo "$all_comments" | jq --argjson pids "$parent_ids" '
      [.[] | select(.id as $id | $pids | index($id)) | {
        id: .id,
        file: .path,
        line: (.original_line // .line),
        body: .body,
        diff_hunk: .diff_hunk
      }]
    ')

    # Build instructions map: parent_id -> user instructions (text after /ai-fixxx, trimmed)
    instructions_json=$(echo "$pending" | jq '
      [.[] | {
        parent_id: .in_reply_to_id,
        instructions: (.body | sub("^/ai-fixxx[[:space:]]*"; ""))
      }]
    ')

    echo "Comments to process:"
    echo "$structured_comments" | jq -c '.[] | {id, file, line}'

    # Fetch the PR diff for context
    echo "Fetching PR diff..."
    diff=$(gh pr diff "$PR_NUMBER" --repo "$REPO")

    # Build prompt
    if [ "$pending_count" -eq 1 ]; then
        # Single pending comment — use focused single-comment prompt
        comment_data=$(echo "$structured_comments" | jq '.[0]')
        user_instr=$(echo "$instructions_json" | jq -r '.[0].instructions')

        if [ -n "$user_instr" ] && [ "$user_instr" != "null" ] && [ "$user_instr" != "" ]; then
            PROMPT="You are AI Fixxxer, an automated code-fix agent. You are given a single review comment from a pull request, the full PR diff for context, and explicit instructions from the user.

The user has replied to this review comment with /ai-fixxx and provided specific instructions. You MUST follow the user's instructions. Do not skip complex changes — the user explicitly asked for this.

## User instructions
${user_instr}

## Rules
- Apply the fix or change as instructed.
- Make ONE git commit with message \`ai-fixxxer: <short description>\`.
- Make targeted changes — do what the user asked, nothing more.
- Do NOT introduce new issues.

## After processing
Write a JSON manifest file to /tmp/ai-fixxxer-results.json containing a single-element array:

If you made a fix:
[{\"comment_id\": <id>, \"action\": \"fix\", \"description\": \"<what you changed>\"}]

If you cannot apply the change (e.g. the request is impossible or would break things):
[{\"comment_id\": <id>, \"action\": \"skip\", \"reason\": \"<why you cannot do this>\"}]

The manifest MUST be valid JSON.

## Review comment:
${comment_data}

## Full PR diff:
${diff}

Now read the file mentioned in the review comment, apply the requested change, commit it, and write the manifest to /tmp/ai-fixxxer-results.json."
        else
            PROMPT="You are AI Fixxxer, an automated code-fix agent. You are given a single review comment from a pull request and the full PR diff for context.

The user has replied to this review comment with /ai-fixxx (no additional instructions). Use your judgment:

- If the comment points to an obvious, unambiguous issue (typo, unused variable, unused import, unhandled error, missing nil check, etc.), fix it.
- If the comment asks for something too complex (refactoring, architectural changes, adding tests), skip it and explain why.
- If the comment's feedback is factually wrong or the code is already correct, reject it and explain why.

## Rules for fixes
- Make ONE git commit with message \`ai-fixxxer: <short description>\`.
- Make minimal, targeted changes — only what the review comment asks for.
- Do NOT introduce new issues.

## After processing
Write a JSON manifest file to /tmp/ai-fixxxer-results.json containing a single-element array:

For fix:
[{\"comment_id\": <id>, \"action\": \"fix\", \"description\": \"<what you changed>\"}]

For skip:
[{\"comment_id\": <id>, \"action\": \"skip\", \"reason\": \"<why this needs human attention>\"}]

For reject:
[{\"comment_id\": <id>, \"action\": \"reject\", \"reason\": \"<why the feedback is incorrect>\"}]

The manifest MUST be valid JSON.

## Review comment:
${comment_data}

## Full PR diff:
${diff}

Now read the file mentioned in the review comment, triage the comment, apply a fix if appropriate, and write the manifest to /tmp/ai-fixxxer-results.json."
        fi
    else
        # Multiple pending comments — build a multi-comment prompt
        # Include per-comment instructions where provided
        comments_section=""
        for i in $(seq 0 $((pending_count - 1))); do
            cdata=$(echo "$structured_comments" | jq ".[$i]")
            cid=$(echo "$cdata" | jq -r '.id')
            cfile=$(echo "$cdata" | jq -r '.file')
            cline=$(echo "$cdata" | jq -r '.line')
            cbody=$(echo "$cdata" | jq -r '.body')
            cdiff=$(echo "$cdata" | jq -r '.diff_hunk')
            cinstr=$(echo "$instructions_json" | jq -r --argjson cid "$cid" '.[] | select(.parent_id == $cid) | .instructions // empty')

            comments_section="${comments_section}
### Comment ID: ${cid}
File: ${cfile}, Line: ${cline}
Review feedback: ${cbody}
User instructions: ${cinstr:-"(none)"}
Diff hunk:
\`\`\`
${cdiff}
\`\`\`
"
        done

        PROMPT="You are AI Fixxxer, an automated code-fix agent. Process each review comment below independently.

For each comment:
- If user instructions are provided, follow them. Do not skip complex changes when the user explicitly asked for them.
- If no user instructions, use your judgment: fix obvious issues (typos, unused vars, unhandled errors), skip complex requests (refactoring, tests), reject wrong feedback.
- Make ONE git commit per fix with message \`ai-fixxxer: <short description>\`.
- Make minimal, targeted changes.
- Do NOT introduce new issues.

## After processing ALL comments
Write a JSON manifest to /tmp/ai-fixxxer-results.json — array with one entry per comment:

For fix: {\"comment_id\": <id>, \"action\": \"fix\", \"description\": \"<what you changed>\"}
For skip: {\"comment_id\": <id>, \"action\": \"skip\", \"reason\": \"<why>\"}
For reject: {\"comment_id\": <id>, \"action\": \"reject\", \"reason\": \"<why>\"}

The manifest MUST be valid JSON. Every comment must appear.

## Comments to process:
${comments_section}

## Full PR diff:
${diff}

Now read the files mentioned in the review comments, process each one, and write the manifest to /tmp/ai-fixxxer-results.json."
    fi

    # Run Claude Code
    echo "Running Claude Code (single-comment mode, ${pending_count} comment(s))..."
    if timeout 10m claude -p --dangerously-skip-permissions "$PROMPT"; then
        echo "Claude Code finished successfully"
    else
        exit_code=$?
        echo "Claude Code exited with code: $exit_code"
    fi

    # Check for manifest
    if [ ! -f "$MANIFEST" ]; then
        echo "WARNING: Claude did not produce a manifest file."
        # Reply to each pending parent comment
        for i in $(seq 0 $((pending_count - 1))); do
            pid=$(echo "$parent_ids" | jq -r ".[$i]")
            post_reply "$pid" "**AI Fixxxer** ran but could not produce results. Check the [workflow run](https://github.com/${REPO}/actions) for details."
        done
        exit 0
    fi

    echo "Manifest contents:"
    cat "$MANIFEST"

    if ! jq empty "$MANIFEST" 2>/dev/null; then
        echo "WARNING: Manifest is not valid JSON."
        for i in $(seq 0 $((pending_count - 1))); do
            pid=$(echo "$parent_ids" | jq -r ".[$i]")
            post_reply "$pid" "**AI Fixxxer** ran but produced an invalid manifest. Check the [workflow run](https://github.com/${REPO}/actions) for details."
        done
        exit 0
    fi

    # Process manifest entries and post replies
    manifest_count=$(jq 'length' "$MANIFEST")
    for i in $(seq 0 $((manifest_count - 1))); do
        entry=$(jq -r ".[$i]" "$MANIFEST")
        comment_id=$(echo "$entry" | jq -r '.comment_id')
        action=$(echo "$entry" | jq -r '.action')

        case "$action" in
            fix)
                description=$(echo "$entry" | jq -r '.description')
                sha=$(git rev-parse --short HEAD 2>/dev/null || true)
                if [ -n "$sha" ]; then
                    post_reply "$comment_id" "Fixed in ${sha}: ${description}"
                else
                    post_reply "$comment_id" "Fixed: ${description}"
                fi
                ;;
            skip)
                reason=$(echo "$entry" | jq -r '.reason')
                post_reply "$comment_id" "**Skipped**: ${reason}"
                ;;
            reject)
                reason=$(echo "$entry" | jq -r '.reason')
                post_reply "$comment_id" "**Not addressed**: ${reason}"
                ;;
            *)
                echo "Unknown action: $action for comment $comment_id"
                post_reply "$comment_id" "**AI Fixxxer** processed the comment but returned an unexpected result. Check the [workflow run](https://github.com/${REPO}/actions) for details."
                ;;
        esac
    done

    echo "AI Fixxxer (single-comment mode) complete: processed ${manifest_count} comment(s)."
    exit 0
fi

# ──────────────────────────────────────────────────────────────────────
# Bulk mode: process all review comments on the PR
# ──────────────────────────────────────────────────────────────────────
echo "AI Fixxxer: Processing PR #${PR_NUMBER} in ${REPO}"

# 1. Fetch all review comments on the PR (exclude replies)
echo "Fetching review comments..."
reviews=$(gh api "repos/${REPO}/pulls/${PR_NUMBER}/comments" --paginate)

structured_reviews=$(echo "$reviews" | jq '[.[] | select(.in_reply_to_id == null) | {
    id: .id,
    file: .path,
    line: (.original_line // .line),
    body: .body,
    diff_hunk: .diff_hunk
}]')

comment_count=$(echo "$structured_reviews" | jq 'length')
echo "Found ${comment_count} review comment(s) (excluding replies)"

if [ "$comment_count" -eq 0 ]; then
    echo "No review comments found. Nothing to fix."
    exit 0
fi

echo "$structured_reviews" | jq -c '.[] | {id, file, line}'

# 2. Fetch the PR diff for context
echo "Fetching PR diff..."
diff=$(gh pr diff "$PR_NUMBER" --repo "$REPO")

# 3. Build prompt for Claude
PROMPT=$(cat <<'PROMPT_HEREDOC'
You are AI Fixxxer, an automated code-fix agent. You are given review comments from a pull request and the full PR diff for context.

Your job is to triage each review comment into one of three categories and act accordingly:

**fix** — The comment points to an obvious, unambiguous issue (typo, unused variable, unused import, unhandled error, missing nil check, etc.). Make the code change.
**skip** — The comment asks for something too complex for an automated tool: refactoring, architectural changes, adding tests, design decisions. Don't touch the code.
**reject** — The comment's feedback is factually wrong or the code is already correct. Don't touch the code.

## Rules for fixes
- Make ONE git commit per fix. Do NOT batch multiple fixes into one commit.
- For each fix: edit the file, then run `git add <specific-files>` and `git commit -m "ai-fixxxer: <short description>"`.
- Make minimal, targeted changes — only what the review comment asks for.
- Do NOT introduce new issues.

## Rules for skip and reject
- Do NOT modify any code.

## After processing ALL comments
Write a JSON manifest file to /tmp/ai-fixxxer-results.json containing an array of objects, one per review comment:

For fix actions:
{"comment_id": <id>, "action": "fix", "description": "<what you changed>"}

For skip actions:
{"comment_id": <id>, "action": "skip", "reason": "<why this needs human attention>"}

For reject actions:
{"comment_id": <id>, "action": "reject", "reason": "<why the feedback is incorrect>"}

The manifest MUST be valid JSON. Write it using a heredoc or echo command. Every comment must appear in the manifest.

PROMPT_HEREDOC
)

PROMPT="${PROMPT}

## Review comments (JSON array with id, file, line, body, diff_hunk):
${structured_reviews}

## Full PR diff:
${diff}

Now read the files mentioned in the review comments, triage each comment, apply fixes (one commit each), and write the manifest to /tmp/ai-fixxxer-results.json."

# 4. Run Claude Code
echo "Running Claude Code..."
if timeout 10m claude -p --dangerously-skip-permissions "$PROMPT"; then
    echo "Claude Code finished successfully"
else
    exit_code=$?
    echo "Claude Code exited with code: $exit_code"
fi

# 5. Check for manifest
if [ ! -f "$MANIFEST" ]; then
    echo "WARNING: Claude did not produce a manifest file."
    gh pr comment "$PR_NUMBER" --repo "$REPO" \
        --body "**AI Fixxxer** ran but did not produce a results manifest. Check the [workflow run](https://github.com/${REPO}/actions) for details."
    exit 0
fi

echo "Manifest contents:"
cat "$MANIFEST"

if ! jq empty "$MANIFEST" 2>/dev/null; then
    echo "WARNING: Manifest is not valid JSON."
    gh pr comment "$PR_NUMBER" --repo "$REPO" \
        --body "**AI Fixxxer** ran but produced an invalid results manifest. Check the [workflow run](https://github.com/${REPO}/actions) for details."
    exit 0
fi

# 6. Collect commit SHAs for fix actions
fix_commits=$(git log --oneline --format="%H %s" | grep "^.\{40\} ai-fixxxer:" || true)

# 7. Post replies to each review comment
total=$(jq 'length' "$MANIFEST")
fixed=0
skipped=0
rejected=0
fix_lines=""
skip_lines=""
reject_lines=""

for i in $(seq 0 $((total - 1))); do
    entry=$(jq -r ".[$i]" "$MANIFEST")
    comment_id=$(echo "$entry" | jq -r '.comment_id')
    action=$(echo "$entry" | jq -r '.action')

    case "$action" in
        fix)
            description=$(echo "$entry" | jq -r '.description')
            sha=$(echo "$fix_commits" | grep -i "$(echo "$description" | head -c 30)" | head -1 | awk '{print $1}' || true)
            if [ -z "$sha" ]; then
                sha=$(echo "$fix_commits" | sed -n "$((fixed + 1))p" | awk '{print $1}' || true)
            fi
            short_sha="${sha:0:7}"

            if [ -n "$sha" ]; then
                reply_body="Fixed in ${short_sha}: ${description}"
            else
                reply_body="Fixed: ${description}"
            fi

            post_reply "$comment_id" "$reply_body"

            fixed=$((fixed + 1))
            fix_lines="${fix_lines}\n- ${description}"
            ;;
        skip)
            reason=$(echo "$entry" | jq -r '.reason')
            skipped=$((skipped + 1))
            skip_lines="${skip_lines}\n- ${reason}"
            ;;
        reject)
            reason=$(echo "$entry" | jq -r '.reason')
            rejected=$((rejected + 1))
            reject_lines="${reject_lines}\n- ${reason}"
            ;;
        *)
            echo "Unknown action: $action for comment $comment_id"
            ;;
    esac
done

# 8. Post summary comment
summary="**AI Fixxxer** processed ${total} review comment(s):"

if [ "$fixed" -gt 0 ]; then
    summary="${summary}\n\n**Fixed (${fixed}):**$(echo -e "$fix_lines")"
fi

if [ "$skipped" -gt 0 ]; then
    summary="${summary}\n\n**Skipped (${skipped})** — needs human attention:$(echo -e "$skip_lines")"
fi

if [ "$rejected" -gt 0 ]; then
    summary="${summary}\n\n**Not addressed (${rejected})** — feedback appears incorrect:$(echo -e "$reject_lines")"
fi

gh pr comment "$PR_NUMBER" --repo "$REPO" --body "$(echo -e "$summary")"

echo "AI Fixxxer complete: ${fixed} fixed, ${skipped} skipped, ${rejected} rejected"
