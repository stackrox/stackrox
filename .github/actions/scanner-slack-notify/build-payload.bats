#!/usr/bin/env bats

setup() {
    TEST_TMP="$(mktemp -d)"
    export TEST_TMP
    ACTION_DIR="$BATS_TEST_DIRNAME"
    FIXTURES="$ACTION_DIR/testdata"

    # Stub gh: serve fixture files instead of calling the GitHub API.
    mkdir -p "$TEST_TMP/bin"
    cat > "$TEST_TMP/bin/gh" <<'EOF'
#!/usr/bin/env bash
path=$2
case "$path" in
*/actions/runs/*/jobs*)
    cat "$GH_STUB_JOBS"
    ;;
*/check-runs/*/annotations*)
    id=$(echo "$path" | sed -E 's#.*/check-runs/([0-9]+)/annotations.*#\1#')
    if [ -f "$GH_STUB_ANNOTATIONS_DIR/$id.json" ]; then
        cat "$GH_STUB_ANNOTATIONS_DIR/$id.json"
    else
        echo '[]'
    fi
    ;;
*)
    echo "gh stub: unexpected call: $*" >&2
    exit 1
    ;;
esac
EOF
    chmod +x "$TEST_TMP/bin/gh"
    PATH="$TEST_TMP/bin:$PATH"

    export GITHUB_REPOSITORY="stackrox/stackrox"
    export GITHUB_RUN_ID="1234567890"
    export GITHUB_WORKFLOW="Scanner versioned vulnerabilities update"
    export GITHUB_EVENT_NAME="schedule"
    export GITHUB_REF_NAME="master"
    export GITHUB_SERVER_URL="https://github.com"
    export MENTION='<!subteam^S04S96AAVND|acs-scanner-primary>'
    export DESCRIPTION='Scanner V4 vulnerability bundle update failed'
    export GH_STUB_JOBS="$FIXTURES/jobs.json"
    export GH_STUB_ANNOTATIONS_DIR="$FIXTURES/annotations"
    export INPUTS_JSON=""
    OUT="$TEST_TMP/payload.json"
}

teardown() {
    rm -rf "$TEST_TMP"
}

@test "produces valid webhook payload with text and blocks" {
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    jq -e '.text and (.blocks | type == "array")' "$OUT"
}

@test "message header carries mention, workflow name, and run link" {
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    header=$(jq -r '.blocks[0].text.text' "$OUT")
    [[ "$header" == *'<!subteam^S04S96AAVND|acs-scanner-primary>'* ]]
    [[ "$header" == *'actions/runs/1234567890'* ]]
    [[ "$header" == *'Scanner versioned vulnerabilities update'* ]]
}

@test "lists failed jobs with matrix values and failed step, skips successful jobs" {
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    all=$(jq -r '[.blocks[].text.text // empty, (.blocks[].elements[]?.text // empty)] | join("\n")' "$OUT")
    [[ "$all" == *'build-and-run (v2, refs/tags/4.9.0)'* ]]
    [[ "$all" == *'Validate bundle'* ]]
    [[ "$all" == *'upload-definitions'* ]]
    [[ "$all" != *'fetch-nvd-feeds'* ]]
}

@test "includes failure annotations, drops exit-code boilerplate and warnings" {
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    all=$(jq -r '[.blocks[].text.text // empty] | join("\n")' "$OUT")
    [[ "$all" == *"source 'osv': file bundles/osv.json.zst contains 0 JSON objects"* ]]
    [[ "$all" != *'Process completed with exit code'* ]]
    [[ "$all" != *'a warning that must not appear'* ]]
}

@test "trigger context line names the event and branch" {
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    ctx=$(jq -r '[.blocks[] | select(.type == "context") | .elements[].text] | join("\n")' "$OUT")
    [[ "$ctx" == *'schedule'* ]]
    [[ "$ctx" == *'master'* ]]
}

@test "escapes slack mrkdwn special characters in annotations" {
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    all=$(jq -r '[.blocks[].text.text // empty] | join("\n")' "$OUT")
    [[ "$all" == *'expected &gt;0'* ]]
    [[ "$all" == *'&lt;bundle&gt; &amp; more'* ]]
    [[ "$all" != *'<bundle>'* ]]
}

@test "truncates long annotations to 500 characters" {
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    ! grep -q 'ENDMARK' "$OUT"
}

@test "renders workflow_dispatch inputs when provided" {
    export GITHUB_EVENT_NAME="workflow_dispatch"
    export INPUTS_JSON='{"stream":"v2","dry_run":"false"}'
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    ctx=$(jq -r '[.blocks[] | select(.type == "context") | .elements[].text] | join("\n")' "$OUT")
    [[ "$ctx" == *'stream=v2'* ]]
    [[ "$ctx" == *'dry_run=false'* ]]
}

@test "accepts literal null inputs json (schedule runs)" {
    export INPUTS_JSON="null"
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    ctx=$(jq -r '[.blocks[] | select(.type == "context") | .elements[].text] | join("\n")' "$OUT")
    [[ "$ctx" != *'Inputs:'* ]]
}

@test "notes when no failed jobs are resolved" {
    export GH_STUB_JOBS="$FIXTURES/jobs-none-failed.json"
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    all=$(jq -r '[.blocks[].text.text // empty] | join("\n")' "$OUT")
    [[ "$all" == *'No failed jobs resolved via the API'* ]]
}

@test "caps annotations at 5 per job" {
    mkdir -p "$TEST_TMP/annotations"
    jq -n '[range(0; 7) | {annotation_level: "failure", message: "error number \(.)"}]' \
        > "$TEST_TMP/annotations/101.json"
    cp "$FIXTURES/annotations/102.json" "$TEST_TMP/annotations/"
    export GH_STUB_ANNOTATIONS_DIR="$TEST_TMP/annotations"
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    all=$(jq -r '[.blocks[].text.text // empty] | join("\n")' "$OUT")
    [[ "$all" == *'error number 4'* ]]
    [[ "$all" != *'error number 5'* ]]
}

@test "caps listed jobs at 10 and notes the remainder" {
    jq -n '{total_count: 12, jobs: [range(0; 12) | {
        id: (200 + .),
        name: "build-and-run (v\(.), refs/tags/4.9.0)",
        conclusion: "failure",
        html_url: "https://github.com/stackrox/stackrox/actions/runs/1234567890/job/\(200 + .)",
        steps: []
    }]}' > "$TEST_TMP/jobs-many.json"
    export GH_STUB_JOBS="$TEST_TMP/jobs-many.json"
    run "$ACTION_DIR/build-payload.sh" "$OUT"
    [ "$status" -eq 0 ]
    count=$(jq '[.blocks[] | select((.text.text // "") | startswith(":x:"))] | length' "$OUT")
    [ "$count" -eq 10 ]
    all=$(jq -r '[.blocks[] | select(.type == "context") | .elements[].text] | join("\n")' "$OUT")
    [[ "$all" == *'2 more failed job(s)'* ]]
}
