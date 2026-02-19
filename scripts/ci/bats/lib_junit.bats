#!/usr/bin/env bats
# shellcheck disable=SC1091

load "../../test_helpers.bats"

function setup() {
    source "${BATS_TEST_DIRNAME}/../lib.sh"
    ARTIFACT_DIR="${BATS_TEST_TMPDIR}"
    junit_dir="$(get_junit_misc_dir)"
}

@test "creates a single junit for a single test" {
    run save_junit_success "UNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="0"'
}

@test "creates multiple junit for multiple tests (different class)" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_success "OtherUNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-OtherUNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="0"'
}

@test "creates multiple junit for multiple tests (same class)" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_success "UNITTest" "Another unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="2"'
    assert_output --partial 'failures="0"'
    run grep -c 'classname="UNITTest"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 2
    run grep -c 'name="A unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
    run grep -c 'name="Another unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "is ok with repeated names" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_success "UNITTest" "Another unit test"
    run save_junit_success "UNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="3"'
    assert_output --partial 'failures="0"'
    run grep -c 'classname="UNITTest"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 3
    run grep -c 'name="A unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 2
    run grep -c 'name="Another unit test"' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "creates a failure junit" {
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="1"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "handles success and failure" {
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_success "UNITTest" "A unit test"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="2"'
    assert_output --partial 'failures="1"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "handles success and failure II" {
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_success "UNITTest" "A unit test"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="4"'
    assert_output --partial 'failures="2"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 2
}

@test "creates a skipped junit" {
    run save_junit_skipped "UNITTest" "A unit test"
    assert_success
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'skipped="1"'
    assert_output --partial 'failures="0"'
    assert_output --partial '<skipped/>'
}

@test "handles success, skipped and failure" {
    run save_junit_success "UNITTest" "A unit test"
    run save_junit_skipped "UNITTest" "A unit test"
    run save_junit_failure "UNITTest" "A unit test" "more failure details"
    run save_junit_skipped "UNITTest" "A unit test"
    run save_junit_success "UNITTest" "A unit test"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="5"'
    assert_output --partial 'failures="1"'
    assert_output --partial 'skipped="2"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 1
}

@test "handles multiline failure details" {
    read -r -d '' details << _EO_DETAILS_ || true
more failure details
other stuff
more failure details
more failure details
_EO_DETAILS_
    run save_junit_failure "UNITTest" "A unit test" "${details}"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="1"'
    run grep -c 'more failure details' "${junit_dir}/junit-UNITTest.xml"
    assert_output 3
}

@test "escapes XML in name - double quote" {
    run save_junit_failure 'UNITTest' '"A unit test"' "nada"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'name="&quot;A unit test&quot;"'
}

@test "escapes XML in name - single quote" {
    run save_junit_failure 'UNITTest' "'A unit test'" "nada"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'name="&#39;A unit test&#39;"'
}

@test "escapes XML in name - <>&" {
    run save_junit_failure 'UNITTest' "A <unit> &test" "nada"
    run cat "${junit_dir}/junit-UNITTest.xml"
    assert_output --partial 'name="A &lt;unit&gt; &amp;test"'
}

@test "junit_contains_failure detects failures from save_junit_failure" {
    run save_junit_failure "UNITTest" "A unit test" "failure details"
    assert_success
    run junit_contains_failure "${ARTIFACT_DIR}"
    assert_success
}

@test "junit_contains_failure returns false for empty directory" {
    run junit_contains_failure "${ARTIFACT_DIR}"
    assert_failure
}

@test "junit_contains_failure returns false for non-existent directory" {
    run junit_contains_failure "${ARTIFACT_DIR}/does-not-exist"
    assert_failure
}

@test "junit_contains_failure returns false for success-only junit" {
    run save_junit_success "UNITTest" "A unit test"
    assert_success
    run junit_contains_failure "${ARTIFACT_DIR}"
    assert_failure
}

@test "junit_contains_failure detects failures with attributes" {
    mkdir -p "${junit_dir}"
    echo '<testsuite><testcase><failure type="error">test</failure></testcase></testsuite>' > "${junit_dir}/test.xml"
    run junit_contains_failure "${ARTIFACT_DIR}"
    assert_success
}

@test "junit_contains_failure detects failures without attributes" {
    mkdir -p "${junit_dir}"
    echo '<testsuite><testcase><failure>test</failure></testcase></testsuite>' > "${junit_dir}/test.xml"
    run junit_contains_failure "${ARTIFACT_DIR}"
    assert_success
}

@test "capture_job_failure_as_junit: job succeeds - no failure record" {
    run capture_job_failure_as_junit \
        "${ARTIFACT_DIR}" \
        "test-job" \
        "success" \
        '{}' \
        "https://github.com/test/repo/actions/runs/123"
    assert_success
    assert_output --partial "no failure record needed"

    # Verify no JUnit files were created (directory may not exist if no files created)
    if [[ -d "${junit_dir}" ]]; then
        run find "${junit_dir}" -name "*.xml"
        assert_output ""
    fi
}

@test "capture_job_failure_as_junit: job fails with existing JUnit failures - skips" {
    # Create existing failure
    save_junit_failure "ExistingTest" "test" "existing failure"

    run capture_job_failure_as_junit \
        "${ARTIFACT_DIR}" \
        "test-job" \
        "failure" \
        '{}' \
        "https://github.com/test/repo/actions/runs/123"
    assert_success
    assert_output --partial "JUnit test failures already exist"

    # Verify only the original failure exists, no test-job file created
    assert [ -f "${junit_dir}/junit-ExistingTest.xml" ]
    assert [ ! -f "${junit_dir}/junit-test-job.xml" ]
    run grep -c 'classname="ExistingTest"' "${junit_dir}/junit-ExistingTest.xml"
    assert_output 1
}

@test "capture_job_failure_as_junit: job fails with failed step - creates specific failure" {
    steps_json='{"my-step":{"outcome":"failure","conclusion":"failure"}}'

    run capture_job_failure_as_junit \
        "${ARTIFACT_DIR}" \
        "test-job" \
        "failure" \
        "$steps_json" \
        "https://github.com/test/repo/actions/runs/123"
    assert_success
    assert_output --partial "Created JUnit failure record for step: my-step"

    # Verify JUnit file exists
    assert [ -f "${junit_dir}/junit-test-job.xml" ]

    # Create expected XML for comparison
    expected="${BATS_TEST_TMPDIR}/expected.xml"
    cat > "$expected" <<'EOF'
<testsuite name="test-job" tests="1" skipped="0" failures="1" errors="0">
        <testcase name="my-step" classname="test-job">
            <failure><![CDATA[Step failed during workflow execution.
Outcome: failure
Conclusion: failure

Check workflow logs for details: https://github.com/test/repo/actions/runs/123]]></failure>
        </testcase>
</testsuite>
EOF

    # Compare actual with expected
    run diff "${junit_dir}/junit-test-job.xml" "$expected"
    assert_success

    # Verify junit_contains_failure detects it
    run junit_contains_failure "${ARTIFACT_DIR}"
    assert_success
}

@test "capture_job_failure_as_junit: job fails without failed step - creates generic failure" {
    steps_json='{"my-step":{"outcome":"success","conclusion":"success"}}'

    run capture_job_failure_as_junit \
        "${ARTIFACT_DIR}" \
        "test-job" \
        "failure" \
        "$steps_json" \
        "https://github.com/test/repo/actions/runs/123"
    assert_success
    assert_output --partial "Created generic JUnit failure record for job: test-job"

    # Verify JUnit file exists
    assert [ -f "${junit_dir}/junit-test-job.xml" ]

    # Create expected XML for comparison
    expected="${BATS_TEST_TMPDIR}/expected-generic.xml"
    cat > "$expected" <<'EOF'
<testsuite name="test-job" tests="1" skipped="0" failures="1" errors="0">
        <testcase name="error" classname="test-job">
            <failure><![CDATA[Job failed without producing JUnit test failures. This typically indicates an infrastructure failure in a step without an id (e.g., docker login, setup step, artifact download).
Check workflow logs: https://github.com/test/repo/actions/runs/123]]></failure>
        </testcase>
</testsuite>
EOF

    # Compare actual with expected
    run diff "${junit_dir}/junit-test-job.xml" "$expected"
    assert_success

    # Verify junit_contains_failure detects it
    run junit_contains_failure "${ARTIFACT_DIR}"
    assert_success
}

@test "capture_job_failure_as_junit: empty steps JSON - creates generic failure" {
    run capture_job_failure_as_junit \
        "${ARTIFACT_DIR}" \
        "test-job" \
        "failure" \
        '{}' \
        "https://github.com/test/repo/actions/runs/123"
    assert_success
    assert_output --partial "Created generic JUnit failure record for job: test-job"

    # Verify the file was created and has correct structure
    assert [ -f "${junit_dir}/junit-test-job.xml" ]
    run cat "${junit_dir}/junit-test-job.xml"
    assert_output --partial 'name="error"'
    assert_output --partial 'failures="1"'
}

@test "capture_job_failure_as_junit: multiple failed steps - uses first one" {
    steps_json='{"step-one":{"outcome":"failure","conclusion":"failure"},"step-two":{"outcome":"failure","conclusion":"failure"}}'

    run capture_job_failure_as_junit \
        "${ARTIFACT_DIR}" \
        "test-job" \
        "failure" \
        "$steps_json" \
        "https://github.com/test/repo/actions/runs/123"
    assert_success
    assert_output --partial "Created JUnit failure record for step: step-one"

    # Verify only step-one is recorded (not step-two)
    run cat "${junit_dir}/junit-test-job.xml"
    assert_output --partial 'name="step-one"'
    refute_output --partial 'name="step-two"'
}

@test "capture_job_failure_as_junit: missing arguments - dies" {
    run capture_job_failure_as_junit \
        "${ARTIFACT_DIR}" \
        "test-job"
    assert_failure
    assert_output --partial "missing args"
}
