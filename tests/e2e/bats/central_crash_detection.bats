#!/usr/bin/env bats
# shellcheck disable=SC1091,SC2034,SC2119,SC2120,SC2153
#
# Unit tests for Central pod crash-state detection in tests/e2e/lib.sh:
#   - check_central_pod_error_state()
#   - wait_for_ready_deployment() Central-specific crash check
#   - wait_for_api() fail-fast on startup crash (CRITICAL regression test)
#
# Test strategy:
#   Place a kubectl stub on PATH that returns fixture JSON per scenario.
#   Override retrying_kubectl to call the stub directly (skip retry wrapper).
#   Override sleep to be instant (no real delays) for deterministic timing.
#   Track kubectl calls in KUBECTL_CALL_LOG for call-order assertions.

load "../../../scripts/test_helpers.bats"

FIXTURE_DIR="${BATS_TEST_DIRNAME}/fixtures"

setup() {
    BIN_DIR="${BATS_TEST_TMPDIR}/bin"
    mkdir -p "${BIN_DIR}"

    KUBECTL_CALL_LOG="${BATS_TEST_TMPDIR}/kubectl-calls.log"
    : > "${KUBECTL_CALL_LOG}"

    # Default fixture: CrashLoopBackOff (overridden per test as needed).
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"
    # Default deployment fixture: never-ready.
    KUBECTL_DEPLOY_READY="false"
    # When set to "true", the kubectl stub's pod-query exits nonzero (simulates
    # API-server failure). Reset per test; default is normal (exit 0).
    KUBECTL_POD_QUERY_FAIL="false"
    # When set to "true", diagnostic kubectl calls (logs, get pod -o json)
    # exit nonzero, simulating stalled/failed diagnostic requests.
    KUBECTL_DIAG_FAIL="false"

    export KUBECTL_CALL_LOG KUBECTL_POD_FIXTURE KUBECTL_DEPLOY_READY KUBECTL_POD_QUERY_FAIL KUBECTL_DIAG_FAIL

    # Stub envsubst (not available in all dev environments; the CI image has it).
    echo '#!/usr/bin/env bash' > "${BIN_DIR}/envsubst"
    chmod +x "${BIN_DIR}/envsubst"

    # Create kubectl stub that dispatches on subcommand.
    cat > "${BIN_DIR}/kubectl" <<'STUB'
#!/usr/bin/env bash
# Append the full command for call-order assertions.
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in
    *"get pods"*|*"get pod"*"-l"*)
        # Configurable failure mode: simulate API-server error.
        if [[ "${KUBECTL_POD_QUERY_FAIL}" == "true" ]]; then
            echo "error: the server is currently unable to handle the request" >&2
            exit 1
        fi
        # Pod listing — return the fixture if set, else empty.
        if [[ -n "${KUBECTL_POD_FIXTURE:-}" && -f "${KUBECTL_POD_FIXTURE}" ]]; then
            cat "${KUBECTL_POD_FIXTURE}"
        else
            echo '{"apiVersion":"v1","kind":"PodList","items":[]}'
        fi
        ;;
    *"get deploy/"*|*"get deployment/"*)
        # Deployment readiness check.
        if [[ "${KUBECTL_DEPLOY_READY}" == "true" ]]; then
            echo '{"status":{"replicas":1,"readyReplicas":1}}'
        else
            echo '{"status":{"replicas":1,"readyReplicas":0}}'
        fi
        ;;
    *"logs"*)
        if [[ "${KUBECTL_DIAG_FAIL}" == "true" ]]; then
            echo "error: request timeout" >&2; exit 1
        fi
        echo "FATAL: migration schema check failed"
        echo "panic: unable to initialize database"
        ;;
    *"get pod"*"-o json"*)
        if [[ "${KUBECTL_DIAG_FAIL}" == "true" ]]; then
            echo "error: request timeout" >&2; exit 1
        fi
        # Single pod state query (diagnostics).
        pod_name=""
        for arg in $*; do
            if [[ "$arg" =~ ^central- ]]; then
                pod_name="$arg"
                break
            fi
        done
        if [[ -n "${KUBECTL_POD_FIXTURE:-}" && -f "${KUBECTL_POD_FIXTURE}" ]]; then
            if [[ -n "$pod_name" ]]; then
                jq --arg name "$pod_name" '.items[] | select(.metadata.name == $name)' "${KUBECTL_POD_FIXTURE}" 2>/dev/null || echo '{}'
            else
                jq '.items[0]' "${KUBECTL_POD_FIXTURE}" 2>/dev/null || echo '{}'
            fi
        else
            echo '{}'
        fi
        ;;
    *)
        echo '{}'
        ;;
esac
STUB
    chmod +x "${BIN_DIR}/kubectl"
    export PATH="${BIN_DIR}:${PATH}"

    # Source lib.sh (which sources scripts/lib.sh for info/die, etc.).
    source "${BATS_TEST_DIRNAME}/../lib.sh"

    # Override retrying_kubectl to use the stub directly (skip retry wrapper).
    retrying_kubectl() {
        # Drain stdin like the real retry-kubectl.sh does.
        cat > /dev/null
        kubectl "$@"
    }
    export -f retrying_kubectl

    # Override sleep to be instant for deterministic tests.
    sleep() {
        : # no-op
    }
    export -f sleep

    # Override curl (used by wait_for_api ping loop) — should not be reached
    # in crash-during-readiness tests, but fail visibly if it is.
    curl() {
        echo "UNEXPECTED: curl should not be reached in readiness-crash tests" >&2
        return 1
    }
    export -f curl
}

# =============================================================================
# check_central_pod_error_state — individual state detection
# =============================================================================

@test "check_central_pod_error_state detects CrashLoopBackOff" {
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"
    run check_central_pod_error_state stackrox
    assert_success   # returns 0 = crash detected
    assert_output --partial "crash state detected"
    assert_output --partial "CrashLoopBackOff"
    # Per-pod diagnostics: pod name and container name are included.
    assert_output --partial "central-abc123-def45/central"
}

@test "check_central_pod_error_state detects current terminated OOMKilled" {
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-oomkilled.json"
    run check_central_pod_error_state stackrox
    assert_success
    assert_output --partial "OOMKilled"
    assert_output --partial "central-abc123-def45/central"
}

@test "check_central_pod_error_state detects current terminated Error" {
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-error.json"
    run check_central_pod_error_state stackrox
    assert_success
    assert_output --partial "terminated"
    assert_output --partial "Error"
    assert_output --partial "central-abc123-def45/central"
}

@test "check_central_pod_error_state reports healthy pod as no crash" {
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-healthy.json"
    run check_central_pod_error_state stackrox
    assert_failure   # returns 1 = no crash
    refute_output --partial "crash state detected"
}

@test "check_central_pod_error_state ignores recovered OOM in lastState (no false positive)" {
    # Regression test for the 3e865ed fix: a pod that OOMed once but recovered
    # to running state must NOT be flagged. Only current .state matters.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-recovered-oom.json"
    run check_central_pod_error_state stackrox
    assert_failure   # returns 1 = no crash
    refute_output --partial "crash state detected"
    refute_output --partial "OOMKilled"
}

@test "check_central_pod_error_state detects crash in second pod (multi-pod)" {
    # During a rolling update, multiple app=central pods coexist.
    # The check must inspect all pods, not just items[0].
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-two-pods-second-crashing.json"
    run check_central_pod_error_state stackrox
    assert_success
    assert_output --partial "crash state detected"
    # Must name the CRASHING pod, not the healthy one.
    assert_output --partial "central-new-xyz78/central"
    assert_output --partial "CrashLoopBackOff"
    # Logs should be fetched for the crashing pod.
    assert_output --partial "logs from pod central-new-xyz78"
}

@test "check_central_pod_error_state handles kubectl failure gracefully" {
    # Simulate kubectl/API-server failure: pod query returns nonzero exit.
    KUBECTL_POD_QUERY_FAIL="true"
    export KUBECTL_POD_QUERY_FAIL
    run check_central_pod_error_state stackrox
    assert_failure   # returns 1 = no crash (safe default on API error)
    refute_output --partial "crash state detected"
}

@test "check_central_pod_error_state handles empty pod list" {
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-empty.json"
    run check_central_pod_error_state stackrox
    assert_failure   # returns 1 = no crash
    refute_output --partial "crash state detected"
}

# =============================================================================
# wait_for_ready_deployment — Central-specific crash check integration
# =============================================================================

@test "wait_for_ready_deployment with Central crash dies with crash message, not timeout" {
    # Central deployment is never ready + pod is crashing.
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"

    # max_seconds=300 is passed as a positional arg (not MAX_WAIT_SECONDS env).
    # run executes in a subshell; die's exit 1 is captured.
    run bash -c '
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        wait_for_ready_deployment stackrox central 300
    ' 2>&1

    assert_failure
    # Must show crash-specific message.
    assert_output --partial "crash state"
    # Must NOT show generic readiness timeout.
    refute_output --partial "wait_for_ready_deployment() timeout"
}

@test "wait_for_ready_deployment with non-Central deployment does not check crash state" {
    # scanner-v4-indexer deployment: never ready, crash check should NOT run.
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"

    # Run with very low timeout so it times out quickly.
    run bash -c '
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        wait_for_ready_deployment stackrox scanner-v4-indexer 1
    ' 2>&1

    assert_failure
    # For non-Central deployments, only the generic timeout fires.
    assert_output --partial "wait_for_ready_deployment() timeout"
    refute_output --partial "crash state"
}

@test "wait_for_ready_deployment generic timeout still works for Central when not crashing" {
    # Central deployment never ready, but pod is healthy (no crash to detect).
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-healthy.json"

    run bash -c '
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        wait_for_ready_deployment stackrox central 1
    ' 2>&1

    assert_failure
    # Falls through to generic timeout since pod is healthy.
    assert_output --partial "wait_for_ready_deployment() timeout"
    refute_output --partial "crash state"
}

# =============================================================================
# wait_for_api — CRITICAL: fail-fast on startup crash before readiness timeout
#
# This is the primary regression test proving the blocker fix works.
# wait_for_api() calls wait_for_ready_deployment() which now checks for
# crash states. A crashing Central must be detected BEFORE the readiness
# timeout, with crash diagnostics instead of a generic timeout message.
# =============================================================================

@test "wait_for_api fails fast on startup crash — crash detected before readiness timeout" {
    # Setup: deployment never becomes ready + pod is in CrashLoopBackOff.
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"

    # timeout(1) enforces a 30-second safety guard: with mocked sleep the
    # crash check fires in <1s, so hitting 30s means a runaway loop. timeout
    # cannot call unexported shell functions, so we wrap in bash -c.
    run timeout 30 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export KUBECTL_DEPLOY_READY='"'${KUBECTL_DEPLOY_READY}'"'
        export PATH='"'${BIN_DIR}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        # Override retrying_kubectl and sleep for deterministic execution.
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        # Set a high MAX_WAIT_SECONDS — the crash check should fire well before.
        MAX_WAIT_SECONDS=300 wait_for_api stackrox
    ' 2>&1

    # Assert intended failure (exit 1 from die), NOT timeout guard (exit 124).
    assert_failure
    [[ "$status" -eq 1 ]]

    # Must print crash diagnostics.
    assert_output --partial "crash state"
    assert_output --partial "CrashLoopBackOff"

    # Must NOT print the generic readiness timeout.
    refute_output --partial "wait_for_ready_deployment() timeout"

    # Call-order assertion: verify crash check ran during readiness wait.
    # The kubectl stub logs every call; "get pods" proves crash check executed.
    assert [ -f "${KUBECTL_CALL_LOG}" ]
    run grep -c "get pods" "${KUBECTL_CALL_LOG}"
    assert_success
    # At least 1 deployment readiness query must have occurred.
    local deploy_checks
    deploy_checks="$(grep -c "get deploy" "${KUBECTL_CALL_LOG}" || true)"
    [[ "$deploy_checks" -ge 1 ]]
    # Fail-fast: should have far fewer than MAX_WAIT_SECONDS/5=60 iterations.
    [[ "$deploy_checks" -le 10 ]]
}

@test "wait_for_api prints crash log markers from affected pod" {
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"

    # 30-second timeout guard; crash detection completes in <1s with mocked sleep.
    run timeout 30 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export KUBECTL_DEPLOY_READY='"'${KUBECTL_DEPLOY_READY}'"'
        export PATH='"'${BIN_DIR}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        MAX_WAIT_SECONDS=300 wait_for_api stackrox
    ' 2>&1

    # Intended exit 1 from die, not timeout guard 124.
    assert_failure
    [[ "$status" -eq 1 ]]
    # Log marker from kubectl stub: diagnostic logs are shown.
    assert_output --partial "migration schema check failed"
}

# =============================================================================
# wait_for_api — ping-loop crash detection (deployment ready, then crash)
#
# Covers the secondary path: Central passes the readiness gate but crashes
# during the ping loop (e.g., post-startup migration failure or OOM under load).
# The every-5th-iteration check in the ping loop should detect it.
# =============================================================================

@test "wait_for_api ping loop detects crash after deployment becomes ready" {
    # Deployment IS ready (passes readiness gate), but pod is crashing.
    KUBECTL_DEPLOY_READY="true"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"

    # 30-second timeout guard; ping-loop crash detection fires in <1s with
    # mocked sleep/curl.
    run timeout 30 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export KUBECTL_DEPLOY_READY='"'${KUBECTL_DEPLOY_READY}'"'
        export PATH='"'${BIN_DIR}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        # curl always fails (API unreachable because pod is crashing).
        curl() { echo "curl: (7) Failed to connect"; return 7; }
        export -f curl
        MAX_WAIT_SECONDS=300 wait_for_api stackrox
    ' 2>&1

    # Intended exit 1 from die/exit, not timeout guard 124.
    assert_failure
    [[ "$status" -eq 1 ]]
    # Crash diagnostics from the ping-loop check.
    assert_output --partial "crash state"
    assert_output --partial "CrashLoopBackOff"
    # Confirm the ping loop was entered (deployment readiness passed).
    assert_output --partial "Deployment central is ready"
}

# =============================================================================
# Terminal pod suppression — precise scope
#
# Only genuinely terminal (phase=Failed/Succeeded) pods have their terminated
# Error/OOMKilled suppressed, and only when a distinct healthy replacement pod
# exists (Running, not deleting, pod Ready condition True, "central" container
# running+ready). Active (Running-phase) pods always detected. Sidecars and
# nonReady pods don't count as healthy replacements.
# =============================================================================

@test "retained terminal pod ignored when genuine healthy replacement exists" {
    # Terminal (Failed-phase) pod + healthy Running replacement with central ready.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-terminated-with-healthy-replacement.json"
    run check_central_pod_error_state stackrox
    assert_failure   # returns 1 = no active crash
    refute_output --partial "crash state detected"
}

@test "active Running pod with terminated Error detected despite healthy peer" {
    # Pod is phase=Running (active, not retained terminal) with terminated
    # central container. Healthy peer exists but must NOT suppress this.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-active-terminated-with-healthy-peer.json"
    run check_central_pod_error_state stackrox
    assert_success   # returns 0 = crash detected on active pod
    assert_output --partial "crash state detected"
    assert_output --partial "central-crashing-abc12/central"
    assert_output --partial "Error"
}

@test "sidecar-ready does not hide failed central container" {
    # Single pod: sidecar running+ready but central container OOMKilled.
    # Sidecar is not named "central" so it must not count as healthy replacement.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-sidecar-healthy-central-failed.json"
    run check_central_pod_error_state stackrox
    assert_success   # returns 0 = crash detected
    assert_output --partial "OOMKilled"
}

@test "active Running pod with OOMKilled detected despite healthy peer" {
    # Active (Running-phase) pod OOMKilled + healthy peer. Must detect.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-active-oomkilled-with-healthy-peer.json"
    run check_central_pod_error_state stackrox
    assert_success
    assert_output --partial "crash state detected"
    assert_output --partial "central-oom-abc12/central"
    assert_output --partial "OOMKilled"
}

@test "nonReady replacement does not suppress terminal pod failure" {
    # Terminal (Failed) pod + Running pod whose Ready condition is False.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-terminal-with-nonready-replacement.json"
    run check_central_pod_error_state stackrox
    assert_success   # returns 0 = crash detected
    assert_output --partial "crash state detected"
    assert_output --partial "Error"
}

@test "deleting replacement does not suppress terminal pod failure" {
    # Terminal (Failed) pod + Running pod that has deletionTimestamp set.
    # A deleting pod must not count as a healthy replacement.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-terminal-with-deleting-replacement.json"
    run check_central_pod_error_state stackrox
    assert_success   # returns 0 = crash detected
    assert_output --partial "crash state detected"
    assert_output --partial "Error"
}

@test "Running-phase OOMKilled pod without replacement still detected" {
    # Single terminated OOMKilled pod with phase=Running, no replacement.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-oomkilled.json"
    run check_central_pod_error_state stackrox
    assert_success   # returns 0 = crash detected
    assert_output --partial "OOMKilled"
}

@test "genuine Failed-phase terminal pod without replacement detected" {
    # Pod with phase=Failed (e.g., evicted), no replacement — strict terminal detection.
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-terminal-failed-phase.json"
    run check_central_pod_error_state stackrox
    assert_success   # returns 0 = crash detected
    assert_output --partial "crash state detected"
    # Pod name and container name appear in diagnostic output.
    assert_output --partial "central-evicted-abc12/central"
    assert_output --partial "terminated"
    assert_output --partial "Error"
}

@test "wait_for_api succeeds with retained terminal pod and genuine healthy replacement" {
    # Deployment ready + terminal old pod + healthy new pod + API responding.
    KUBECTL_DEPLOY_READY="true"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-terminated-with-healthy-replacement.json"

    run timeout 30 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export KUBECTL_DEPLOY_READY='"'${KUBECTL_DEPLOY_READY}'"'
        export PATH='"'${BIN_DIR}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        curl() { echo '"'"'{"status":"ok"}'"'"'; return 0; }
        export -f curl
        MAX_WAIT_SECONDS=300 wait_for_api stackrox
    ' 2>&1

    assert_success
    refute_output --partial "crash state"
}

# =============================================================================
# Request-timeout flag and diagnostic failure resilience
# =============================================================================

@test "check_central_pod_error_state passes request-timeout to kubectl" {
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"
    run check_central_pod_error_state stackrox
    assert_success
    run grep -c "request-timeout" "${KUBECTL_CALL_LOG}"
    assert_success
    [[ "${output}" -ge 1 ]]
}

@test "crash detected despite diagnostic call failure" {
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"
    KUBECTL_DIAG_FAIL="true"
    export KUBECTL_DIAG_FAIL
    run check_central_pod_error_state stackrox
    assert_success
    assert_output --partial "crash state detected"
    assert_output --partial "CrashLoopBackOff"
}

# =============================================================================
# Genuine blocking-call deadline tests
#
# These tests use a kubectl stub that actually blocks (real sleep, NOT the
# mocked no-op sleep) to prove that the production timeout mechanisms
# (CRASH_CHECK_QUERY_TIMEOUT and CRASH_CHECK_DIAG_TIMEOUT via GNU timeout(1))
# enforce finite deadlines. The test-scoped deadline is narrowed to 2s via
# the configurable env vars so tests complete quickly.
# =============================================================================

@test "blocking pod query bounded by CRASH_CHECK_QUERY_TIMEOUT" {
    # Create a kubectl stub that blocks on pod queries using real sleep.
    # The test overrides retrying_kubectl to call this directly (no retry
    # wrapper, so the timeout wraps the single blocking call).
    local HANG_BIN="${BATS_TEST_TMPDIR}/hang-bin"
    mkdir -p "${HANG_BIN}"
    cat > "${HANG_BIN}/kubectl" <<'HANGSTUB'
#!/usr/bin/env bash
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in
    *"get pods"*)
        # Block for 60s — longer than any test deadline.
        /usr/bin/sleep 60
        ;;
    *) echo '{}' ;;
esac
HANGSTUB
    chmod +x "${HANG_BIN}/kubectl"

    # Use 2s query timeout for fast test. Source lib.sh for real function.
    CRASH_CHECK_QUERY_TIMEOUT=2
    export CRASH_CHECK_QUERY_TIMEOUT PATH="${HANG_BIN}:${PATH}"

    # Override retrying_kubectl to call kubectl directly (skip retry wrapper
    # which would multiply the hang). The timeout in check_central_pod_error_state
    # wraps retrying_kubectl.
    retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
    export -f retrying_kubectl

    local start_time end_time elapsed
    start_time="$(date +%s)"
    run check_central_pod_error_state stackrox
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))

    # Function must return (not hang), within the deadline + margin.
    assert_failure   # returns 1 = no crash (query timed out, pod_json empty)
    refute_output --partial "crash state detected"
    # Lower bound: deadline actually enforced (not instant skip). Upper bound: not 60s hang.
    [[ "$elapsed" -ge 1 ]]
    [[ "$elapsed" -le 5 ]]
    # The blocked pod-query call was actually attempted (logged before sleep).
    run grep "get pods" "${KUBECTL_CALL_LOG}"
    assert_success
}

@test "blocking diagnostic log call bounded by CRASH_CHECK_DIAG_TIMEOUT" {
    local HANG_BIN="${BATS_TEST_TMPDIR}/hang-bin"
    mkdir -p "${HANG_BIN}"
    cat > "${HANG_BIN}/kubectl" <<'HANGSTUB'
#!/usr/bin/env bash
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in
    *"get pods"*)
        # Pod query succeeds (return crashloop fixture).
        cat "${KUBECTL_POD_FIXTURE}"
        ;;
    *"logs"*)
        # Block on logs — simulates stalled API.
        /usr/bin/sleep 60
        ;;
    *"get pod"*"-o json"*)
        echo '{}'
        ;;
    *) echo '{}' ;;
esac
HANGSTUB
    chmod +x "${HANG_BIN}/kubectl"

    CRASH_CHECK_DIAG_TIMEOUT=2
    CRASH_CHECK_QUERY_TIMEOUT=5
    export CRASH_CHECK_DIAG_TIMEOUT CRASH_CHECK_QUERY_TIMEOUT
    export PATH="${HANG_BIN}:${PATH}"
    retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
    export -f retrying_kubectl

    local start_time end_time elapsed
    start_time="$(date +%s)"
    run check_central_pod_error_state stackrox
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))

    # Crash STILL detected despite log call timing out.
    assert_success   # returns 0 = crash detected
    assert_output --partial "crash state detected"
    assert_output --partial "CrashLoopBackOff"
    # Lower bound: deadline actually enforced (not instant skip). Upper bound: not 60s hang.
    [[ "$elapsed" -ge 1 ]]
    [[ "$elapsed" -le 5 ]]
    # The blocked logs call was actually attempted.
    run grep "logs" "${KUBECTL_CALL_LOG}"
    assert_success
}

@test "blocking diagnostic get-pod call bounded by CRASH_CHECK_DIAG_TIMEOUT" {
    local HANG_BIN="${BATS_TEST_TMPDIR}/hang-bin"
    mkdir -p "${HANG_BIN}"
    cat > "${HANG_BIN}/kubectl" <<'HANGSTUB'
#!/usr/bin/env bash
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in
    *"get pods"*)
        cat "${KUBECTL_POD_FIXTURE}"
        ;;
    *"logs"*)
        echo "FATAL: migration failed"
        ;;
    *"get pod"*"-o json"*)
        # Block on single-pod state query.
        /usr/bin/sleep 60
        ;;
    *) echo '{}' ;;
esac
HANGSTUB
    chmod +x "${HANG_BIN}/kubectl"

    CRASH_CHECK_DIAG_TIMEOUT=2
    CRASH_CHECK_QUERY_TIMEOUT=5
    export CRASH_CHECK_DIAG_TIMEOUT CRASH_CHECK_QUERY_TIMEOUT
    export PATH="${HANG_BIN}:${PATH}"
    retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
    export -f retrying_kubectl

    local start_time end_time elapsed
    start_time="$(date +%s)"
    run check_central_pod_error_state stackrox
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))

    # Crash detected despite get-pod hanging.
    assert_success
    assert_output --partial "crash state detected"
    assert_output --partial "CrashLoopBackOff"
    # Lower bound: deadline actually enforced. Upper bound: not 60s hang.
    [[ "$elapsed" -ge 1 ]]
    [[ "$elapsed" -le 5 ]]
    # The blocked get-pod call was actually attempted.
    run grep "get pod" "${KUBECTL_CALL_LOG}"
    assert_success
}

# =============================================================================
# Budget propagation — caller-supplied remaining time caps crash check
# =============================================================================

@test "zero or negative budget makes no kubectl calls" {
    : > "${KUBECTL_CALL_LOG}"
    run check_central_pod_error_state stackrox 0
    assert_failure
    run cat "${KUBECTL_CALL_LOG}"
    assert_output ""
    # Negative budget also returns immediately.
    : > "${KUBECTL_CALL_LOG}"
    run check_central_pod_error_state stackrox -5
    assert_failure
    run cat "${KUBECTL_CALL_LOG}"
    assert_output ""
}

@test "blocked query capped by caller budget with call marker" {
    local HANG_BIN="${BATS_TEST_TMPDIR}/hang-bin"
    mkdir -p "${HANG_BIN}"
    cat > "${HANG_BIN}/kubectl" <<'HANGSTUB'
#!/usr/bin/env bash
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in *"get pods"*) /usr/bin/sleep 60 ;; *) echo '{}' ;; esac
HANGSTUB
    chmod +x "${HANG_BIN}/kubectl"
    export PATH="${HANG_BIN}:${PATH}"
    retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
    export -f retrying_kubectl
    local start_time end_time elapsed
    start_time="$(date +%s)"
    run check_central_pod_error_state stackrox 3
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))
    assert_failure
    [[ "$elapsed" -ge 1 ]] && [[ "$elapsed" -le 6 ]]
    run grep "get pods" "${KUBECTL_CALL_LOG}"
    assert_success
}

@test "blocked logs exhausts cumulative budget: no get-pod or second-pod diagnostics" {
    # Generate a fixture with TWO crashing pods (both CrashLoopBackOff) so
    # affected_pods has 2 entries. First pod's logs block → budget exhausted
    # → get-pod on first pod skipped, all of second pod's diagnostics skipped.
    local TWO_CRASH="${BATS_TEST_TMPDIR}/two-crash.json"
    jq '.items[0].status.containerStatuses[0].state = {"waiting":{"reason":"CrashLoopBackOff"}} |
        .items[0].status.containerStatuses[0].ready = false |
        .items[0].status.containerStatuses[0].restartCount = 3 |
        .items[0].status.conditions = [] |
        .items[1].status.containerStatuses[0].state = {"waiting":{"reason":"CrashLoopBackOff"}} |
        .items[1].status.containerStatuses[0].ready = false |
        .items[1].status.containerStatuses[0].restartCount = 5' \
      "${FIXTURE_DIR}/central-two-pods-second-crashing.json" > "${TWO_CRASH}"
    KUBECTL_POD_FIXTURE="${TWO_CRASH}"
    local HANG_BIN="${BATS_TEST_TMPDIR}/hang-bin"
    mkdir -p "${HANG_BIN}"
    cat > "${HANG_BIN}/kubectl" <<'HANGSTUB'
#!/usr/bin/env bash
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in
    *"get pods"*) cat "${KUBECTL_POD_FIXTURE}" ;;
    *"logs"*) /usr/bin/sleep 60 ;;
    *"get pod"*"-o json"*) echo '{}' ;;
    *) echo '{}' ;;
esac
HANGSTUB
    chmod +x "${HANG_BIN}/kubectl"
    export PATH="${HANG_BIN}:${PATH}"
    retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
    export -f retrying_kubectl
    local start_time end_time elapsed
    start_time="$(date +%s)"
    run check_central_pod_error_state stackrox 3
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))
    assert_success
    assert_output --partial "crash state detected"
    assert_output --partial "CrashLoopBackOff"
    assert_output --partial "budget exhausted"
    [[ "$elapsed" -ge 1 ]] && [[ "$elapsed" -le 8 ]]
    # Only ONE logs call attempted (first pod, blocked).
    local logs_count
    logs_count="$(grep -c 'logs' "${KUBECTL_CALL_LOG}" || true)"
    [[ "$logs_count" -eq 1 ]]
    # No get-pod diagnostics at all.
    local getpod_diag_count
    getpod_diag_count="$(grep -c 'get pod -n' "${KUBECTL_CALL_LOG}" || true)"
    [[ "$getpod_diag_count" -eq 0 ]]
}

@test "wait_for_ready_deployment near-deadline passes remaining to helper and times out" {
    # Deployment never ready. Stub kubectl: deployment query instant,
    # crash-check pod query blocks 60s. max_seconds=3. The crash check
    # should be capped to remaining ~3s, then deployment timeout fires.
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"
    local HANG_BIN="${BATS_TEST_TMPDIR}/hang-bin"
    mkdir -p "${HANG_BIN}"
    cat > "${HANG_BIN}/kubectl" <<'HANGSTUB'
#!/usr/bin/env bash
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in
    *"get deploy"*) echo '{"status":{"replicas":1,"readyReplicas":0}}' ;;
    *"get pods"*) /usr/bin/sleep 60 ;;
    *"get pod"*) echo '{}' ;;
    *"logs"*) echo "log" ;;
    *) echo '{}' ;;
esac
HANGSTUB
    chmod +x "${HANG_BIN}/kubectl"
    local start_time end_time elapsed
    start_time="$(date +%s)"
    # 15s outer safety guard.
    run timeout 15 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export PATH='"'${HANG_BIN}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        wait_for_ready_deployment stackrox central 3
    ' 2>&1
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))
    assert_failure
    [[ "$status" -eq 1 ]]
    assert_output --partial "wait_for_ready_deployment() timeout"
    refute_output --partial "crash state detected"
    run grep "get pods" "${KUBECTL_CALL_LOG}"
    assert_success
    # Completed within 3s budget + margin, not hung 60s or outer 15s.
    [[ "$elapsed" -ge 1 ]]
    [[ "$elapsed" -le 8 ]]
}

@test "wait_for_ready_deployment expired deadline skips helper entirely" {
    # Use controlled clock: sleep is no-op so iterations are instant.
    # After 5 iterations the crash check would fire, but max_seconds=0
    # means deadline already expired → helper skipped.
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"
    run timeout 10 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export KUBECTL_DEPLOY_READY="false"
        export PATH='"'${BIN_DIR}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        wait_for_ready_deployment stackrox central 0
    ' 2>&1
    assert_failure
    [[ "$status" -eq 1 ]]
    assert_output --partial "wait_for_ready_deployment() timeout"
    # No crash-check pod query made (helper skipped because deadline expired).
    local pods_count
    pods_count="$(grep -c 'get pods' "${KUBECTL_CALL_LOG}" || true)"
    [[ "$pods_count" -eq 0 ]]
}

@test "exact deadline boundary: >= fires at max_seconds without extra poll cycle" {
    # Real wall-clock test: max_seconds=1 with no-op sleep. Deployment never
    # ready. The >= comparison fires after 1 second of real elapsed time.
    # Count deployment queries: with >=, the loop terminates as soon as
    # elapsed reaches 1s. With old >, it would allow another full cycle
    # (query + sleep + query) before timing out, producing more queries.
    KUBECTL_DEPLOY_READY="false"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-healthy.json"

    local start_time end_time elapsed
    start_time="$(date +%s)"
    run timeout 10 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export PATH='"'${BIN_DIR}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        wait_for_ready_deployment stackrox central 1
    ' 2>&1
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))

    assert_failure
    [[ "$status" -eq 1 ]]
    assert_output --partial "wait_for_ready_deployment() timeout"
    refute_output --partial "crash state"
    # Elapsed must be close to 1s (the deadline), not 10s (outer guard).
    [[ "$elapsed" -ge 1 ]]
    [[ "$elapsed" -le 4 ]]
}

@test "exact deadline boundary: nonCentral deployment times out at >= boundary" {
    # Simple guard: nonCentral deployment also uses >= for timeout.
    KUBECTL_DEPLOY_READY="false"
    run timeout 10 bash -c '
        export PATH='"'${BIN_DIR}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        wait_for_ready_deployment stackrox scanner-v4-indexer 0
    ' 2>&1
    assert_failure
    [[ "$status" -eq 1 ]]
    assert_output --partial "wait_for_ready_deployment() timeout"
    refute_output --partial "crash state"
}

@test "wait_for_api budget stops loop on own deadline not after 149 more curls" {
    # Deployment ready. Stub: pod query blocks 60s (will be capped by
    # budget). Curl fails fast. Budget is 3s. After crash check times out
    # at budget, API deadline check breaks loop — NOT 149 more curl calls.
    KUBECTL_DEPLOY_READY="true"
    KUBECTL_POD_FIXTURE="${FIXTURE_DIR}/central-crashloop.json"
    local HANG_BIN="${BATS_TEST_TMPDIR}/hang-bin"
    mkdir -p "${HANG_BIN}"
    cat > "${HANG_BIN}/kubectl" <<'HANGSTUB'
#!/usr/bin/env bash
echo "kubectl $*" >> "${KUBECTL_CALL_LOG}"
case "$*" in
    *"get deploy"*) echo '{"status":{"replicas":1,"readyReplicas":1}}' ;;
    *"get pods"*) /usr/bin/sleep 60 ;;
    *"logs"*) echo "log" ;;
    *"get pod"*) echo '{}' ;;
    *) echo '{}' ;;
esac
HANGSTUB
    chmod +x "${HANG_BIN}/kubectl"
    local start_time end_time elapsed
    start_time="$(date +%s)"
    # 15s outer safety guard.
    run timeout 15 bash -c '
        export KUBECTL_CALL_LOG='"'${KUBECTL_CALL_LOG}'"'
        export KUBECTL_POD_FIXTURE='"'${KUBECTL_POD_FIXTURE}'"'
        export KUBECTL_DEPLOY_READY="true"
        export PATH='"'${HANG_BIN}:${PATH}'"'
        source "'"${BATS_TEST_DIRNAME}"'/../lib.sh"
        retrying_kubectl() { cat > /dev/null; kubectl "$@"; }
        export -f retrying_kubectl
        sleep() { :; }
        export -f sleep
        # curl stub logs each call to KUBECTL_CALL_LOG for counting.
        curl() { echo "ping-curl" >> "${KUBECTL_CALL_LOG}"; echo "curl: (7) Failed"; return 7; }
        export -f curl
        CRASH_CHECK_API_PING_BUDGET=3 MAX_WAIT_SECONDS=300 wait_for_api stackrox
    ' 2>&1
    end_time="$(date +%s)"
    elapsed=$(( end_time - start_time ))
    assert_failure
    [[ "$status" -eq 1 ]]
    assert_output --partial "deadline"
    refute_output --partial "crash state detected"
    run grep "get pods" "${KUBECTL_CALL_LOG}"
    assert_success
    # Curl calls actually logged; budget stops the loop well before 149.
    local curl_count
    curl_count="$(grep -c 'ping-curl' "${KUBECTL_CALL_LOG}" || true)"
    [[ "$curl_count" -ge 1 ]]
    [[ "$curl_count" -le 5 ]]
    # Completed within 3s budget + margin, not outer 15s.
    [[ "$elapsed" -ge 1 ]]
    [[ "$elapsed" -le 8 ]]
}
