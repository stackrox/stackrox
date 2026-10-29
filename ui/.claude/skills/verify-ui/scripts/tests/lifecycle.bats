#!/usr/bin/env bats

setup() {
    scripts_dir="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"
    test_dir="$(mktemp -d)"
    export VERIFY_UI_HOME="${test_dir}/home"
    export MOCK_CALLS="${test_dir}/calls"
    export MOCK_PLATFORM_DIR="$(cd "${scripts_dir}/../../../../apps/platform" && pwd)"
    export MOCK_PORT_OWNED=0
    mkdir -p "${test_dir}/bin" "${VERIFY_UI_HOME}/state/tokens" "${VERIFY_UI_HOME}/state/active-runs"
    printf 'UI_START_TARGET=https://central.example\nCENTRAL_USER=admin\nINFRA_CLUSTER=\n' > "${VERIFY_UI_HOME}/state/backend.env"
    printf 'password\n' > "${VERIFY_UI_HOME}/state/admin-password"
    cat > "${test_dir}/bin/curl" <<'EOF'
#!/usr/bin/env bash
for arg in "$@"; do
    case "${arg}" in
    */v1/apitokens/generate)
        printf '{"token":"secret","metadata":{"id":"token-%s"}}\n' "$$"
        exit 0
        ;;
    */v1/apitokens/revoke/*)
        printf '%s\n' "${arg}" >> "${MOCK_CALLS}"
        exit 0
        ;;
    */v1/featureflags)
        printf '{"featureFlags":[]}\n'
        exit 0
        ;;
    */v1/metadata)
        printf '{"version":"test"}\n'
        exit 0
        ;;
    esac
done
exit 1
EOF
    cat > "${test_dir}/bin/lsof" <<'EOF'
#!/usr/bin/env bash
case " $* " in
*" -d cwd "*) printf 'n%s\n' "${MOCK_CWD:-${MOCK_PLATFORM_DIR}}" ;;
*) [[ "${MOCK_PORT_OWNED}" == 1 ]] ;;
esac
EOF
    cat > "${test_dir}/bin/npx" <<'EOF'
#!/usr/bin/env bash
sleep "${MOCK_RUN_SLEEP:-0}"
printf '{"assertions":{"proof":1},"errors":[]}\n' > "${CYPRESS_VERIFY_EVIDENCE_DIR}/checks.json"
exit "${MOCK_RUN_EXIT:-0}"
EOF
    cat > "${test_dir}/bin/ps" <<'EOF'
#!/usr/bin/env bash
[[ "${MOCK_PS_DENY:-0}" == 0 ]] || exit 1
kill -0 "$2" 2>/dev/null || exit 1
printf 'Thu Jan  1 00:00:00 1970\n'
EOF
    chmod +x "${test_dir}/bin/"*
    export PATH="${test_dir}/bin:${PATH}"
}

teardown() {
    rm -rf "${test_dir}"
}

@test "cleanup revokes only token IDs recorded for the connected Central" {
    mkdir -p "${VERIFY_UI_HOME}/state/tokens/own" "${VERIFY_UI_HOME}/state/tokens/other"
    printf 'own-id\n' > "${VERIFY_UI_HOME}/state/tokens/own/id"
    printf 'https://central.example\n' > "${VERIFY_UI_HOME}/state/tokens/own/target"
    printf 'other-id\n' > "${VERIFY_UI_HOME}/state/tokens/other/id"
    printf 'https://other.example\n' > "${VERIFY_UI_HOME}/state/tokens/other/target"

    run "${scripts_dir}/cleanup.sh"

    [ "${status}" -eq 0 ]
    [ "$(cat "${MOCK_CALLS}")" = 'https://central.example/v1/apitokens/revoke/own-id' ]
    [ ! -e "${VERIFY_UI_HOME}/state/tokens/own" ]
    [ -d "${VERIFY_UI_HOME}/state/tokens/other" ]
}

@test "launch and cleanup refuse a live pid that does not own the configured port" {
    printf '%s\n' "$$" > "${VERIFY_UI_HOME}/state/vite.pid"

    run "${scripts_dir}/launch.sh"
    [ "${status}" -ne 0 ]
    [[ "${output}" == *"cannot verify that dev server pid"* ]]

    run "${scripts_dir}/cleanup.sh"
    [ "${status}" -ne 0 ]
    [[ "${output}" == *"cannot verify that dev server pid"* ]]
    [ ! -e "${MOCK_CALLS}" ]
    kill -0 "$$"
}

@test "launch refuses a live pid from another checkout" {
    export MOCK_PORT_OWNED=1 MOCK_CWD="${test_dir}/other-checkout"
    printf '%s\n' "$$" > "${VERIFY_UI_HOME}/state/vite.pid"

    run "${scripts_dir}/launch.sh"

    [ "${status}" -ne 0 ]
    [[ "${output}" == *"cannot verify that dev server pid"* ]]
    [[ "${output}" == *"runs from"* ]]
    kill -0 "$$"
}

@test "concurrent drives revoke their own tokens and block cleanup until both finish" {
    export MOCK_PORT_OWNED=1
    export MOCK_PS_DENY=1
    printf '%s\n' "$$" > "${VERIFY_UI_HOME}/state/vite.pid"
    printf 'describe("proof", () => {});\n' > "${test_dir}/proof.cy.js"

    MOCK_RUN_SLEEP=2 "${scripts_dir}/drive.sh" "${test_dir}/proof.cy.js" > "${test_dir}/drive1.log" 2>&1 &
    first=$!
    MOCK_RUN_SLEEP=5 "${scripts_dir}/drive.sh" "${test_dir}/proof.cy.js" > "${test_dir}/drive2.log" 2>&1 &
    second=$!

    for _ in $(seq 1 50); do
        [ "$(find "${VERIFY_UI_HOME}/state/active-runs" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" -eq 2 ] && break
        sleep 0.1
    done
    [ "$(find "${VERIFY_UI_HOME}/state/active-runs" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')" -eq 2 ] || {
        echo "# first: $(cat "${test_dir}/drive1.log")"
        echo "# second: $(cat "${test_dir}/drive2.log")"
        false
    }
    run "${scripts_dir}/cleanup.sh" --backend
    [ "${status}" -ne 0 ]
    [[ "${output}" == *"active drive.sh run(s)"* ]]

    wait "${first}"
    [ "$(wc -l < "${MOCK_CALLS}" | tr -d ' ')" -eq 1 ]
    run "${scripts_dir}/cleanup.sh"
    [ "${status}" -ne 0 ]
    [[ "${output}" == *"active drive.sh run(s)"* ]]

    wait "${second}"
    [ "$(wc -l < "${MOCK_CALLS}" | tr -d ' ')" -eq 2 ]
    [ "$(sort -u "${MOCK_CALLS}" | wc -l | tr -d ' ')" -eq 2 ]
    [ "$(tail -1 "${test_dir}/drive1.log")" != "$(tail -1 "${test_dir}/drive2.log")" ]
    [ -f "$(tail -1 "${test_dir}/drive1.log")/summary.txt" ]
    [ -f "$(tail -1 "${test_dir}/drive2.log")/summary.txt" ]
    [ -f "${VERIFY_UI_HOME}/state/vite.pid" ]
    kill -0 "$$"
}

@test "failed Cypress run revokes its token and clears active state" {
    export MOCK_PORT_OWNED=1 MOCK_RUN_EXIT=1
    printf '%s\n' "$$" > "${VERIFY_UI_HOME}/state/vite.pid"
    printf 'describe("proof", () => {});\n' > "${test_dir}/proof.cy.js"

    run "${scripts_dir}/drive.sh" "${test_dir}/proof.cy.js"

    [ "${status}" -eq 1 ]
    [ "$(wc -l < "${MOCK_CALLS}" | tr -d ' ')" -eq 1 ]
    [ -z "$(find "${VERIFY_UI_HOME}/state/active-runs" -mindepth 1 -maxdepth 1 -type d)" ]
    [ -z "$(find "${VERIFY_UI_HOME}/state/tokens" -mindepth 1 -maxdepth 1 -type d)" ]
}
