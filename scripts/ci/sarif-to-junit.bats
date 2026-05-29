#!/usr/bin/env bats

# Allow to run the tests locally provided that bats-helpers are installed in $HOME/bats-core
bats_helpers_root="${HOME}/bats-core"
if [[ ! -f "${bats_helpers_root}/bats-support/load.bash" ]]; then
  # Location of bats-helpers in the CI image
  bats_helpers_root="/usr/lib/node_modules"
fi
load "${bats_helpers_root}/bats-support/load.bash"
load "${bats_helpers_root}/bats-assert/load.bash"

function setup() {
    source "${BATS_TEST_DIRNAME}/lib.sh"
    ARTIFACT_DIR="${BATS_TEST_TMPDIR}/junit-reports"
    mkdir -p "${ARTIFACT_DIR}"
    export ARTIFACT_DIR
}

function teardown() {
    rm -rf "${ARTIFACT_DIR}"
}

@test "missing arguments" {
    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh"
    assert_failure 1
    assert_output --partial 'Usage:'
}

@test "sarif file not found" {
    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh" nonexistent.sarif test-image:latest
    assert_failure 1
    assert_output --partial 'SARIF file not found'
}

@test "empty sarif report creates success record" {
    local sarif_file="${BATS_TEST_TMPDIR}/empty.sarif"
    cat > "${sarif_file}" <<'EOF'
{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "roxctl"
        }
      },
      "results": []
    }
  ]
}
EOF

    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh" "${sarif_file}" "test-image:latest"
    assert_success
    assert_output --partial 'No vulnerabilities found'

    # Verify success JUnit record was created (one file per image)
    local junit_file="${ARTIFACT_DIR}/junit-misc/junit-test-image_latest.xml"
    assert [ -f "${junit_file}" ]

    # Verify structure: testsuite name is image name, no failures
    run cat "${junit_file}"
    assert_output --partial '<testsuite name="test-image:latest"'
    assert_output --partial 'tests="1"'
    assert_output --partial 'failures="0"'
    assert_output --partial '<testcase name="vulnerability-scan" classname="vulnerability-scan">'
}

@test "converts sarif violations to junit failures" {
    local sarif_file="${BATS_TEST_TMPDIR}/violations.sarif"
    cat > "${sarif_file}" <<'EOF'
{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "roxctl",
          "rules": []
        }
      },
      "results": [
        {
          "ruleId": "CVE-2024-1234_nginx_1.19.0",
          "level": "error",
          "message": {
            "text": "Critical vulnerability in nginx"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "quay.io/stackrox-io/main:4.6.0"
                }
              }
            }
          ]
        },
        {
          "ruleId": "CVE-2024-5678_openssl_1.1.1",
          "level": "warning",
          "message": {
            "text": "Important vulnerability in openssl"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "quay.io/stackrox-io/main:4.6.0"
                }
              }
            }
          ]
        }
      ]
    }
  ]
}
EOF

    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh" "${sarif_file}" "quay.io/stackrox-io/main:4.6.0"
    assert_success
    assert_output --partial 'Converted 2 vulnerabilities'

    # One JUnit file per image (testsuite), contains all vulnerabilities
    local junit_file="${ARTIFACT_DIR}/junit-misc/junit-quay.io_stackrox-io_main_4.6.0.xml"
    assert [ -f "${junit_file}" ]

    # Verify testsuite structure: name is image, 2 tests, 2 failures
    run cat "${junit_file}"
    assert_output --partial '<testsuite name="quay.io/stackrox-io/main:4.6.0"'
    assert_output --partial 'tests="2"'
    assert_output --partial 'failures="2"'

    # Verify first testcase: classname is CVE, name is component_version
    assert_output --partial '<testcase name="nginx_1.19.0" classname="CVE-2024-1234">'
    assert_output --partial 'Critical vulnerability in nginx'
    assert_output --partial 'Severity: error'

    # Verify second testcase
    assert_output --partial '<testcase name="openssl_1.1.1" classname="CVE-2024-5678">'
    assert_output --partial 'Important vulnerability in openssl'
    assert_output --partial 'Severity: warning'
}

@test "handles sarif results without locations" {
    local sarif_file="${BATS_TEST_TMPDIR}/no-location.sarif"
    cat > "${sarif_file}" <<'EOF'
{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "roxctl"
        }
      },
      "results": [
        {
          "ruleId": "CVE-2024-9999_test_1.0.0",
          "level": "error",
          "message": {
            "text": "Test vulnerability"
          }
        }
      ]
    }
  ]
}
EOF

    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh" "${sarif_file}" "test-image:latest"
    assert_success
    assert_output --partial 'Converted 1 vulnerabilities'

    # Should still create JUnit record even without location
    local junit_file="${ARTIFACT_DIR}/junit-misc/junit-test-image_latest.xml"
    assert [ -f "${junit_file}" ]

    # Verify structure
    run cat "${junit_file}"
    assert_output --partial '<testsuite name="test-image:latest"'
    assert_output --partial '<testcase name="test_1.0.0" classname="CVE-2024-9999">'
}

@test "parses CVE from ruleId correctly" {
    local sarif_file="${BATS_TEST_TMPDIR}/parse-test.sarif"
    cat > "${sarif_file}" <<'EOF'
{
  "version": "2.1.0",
  "runs": [{
    "tool": {"driver": {"name": "roxctl"}},
    "results": [
      {
        "ruleId": "CVE-2024-1234_nginx_1.19.0",
        "level": "error",
        "message": {"text": "Test with proper CVE format"}
      },
      {
        "ruleId": "GHSA-abcd-1234_golang_1.20.0",
        "level": "warning",
        "message": {"text": "Test with GHSA format"}
      },
      {
        "ruleId": "no-underscore-format",
        "level": "note",
        "message": {"text": "Test without underscore"}
      }
    ]
  }]
}
EOF

    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh" "${sarif_file}" "test:latest"
    assert_success

    local junit_file="${ARTIFACT_DIR}/junit-misc/junit-test_latest.xml"
    run cat "${junit_file}"

    # First: CVE-2024-1234 as classname, nginx_1.19.0 as name
    assert_output --partial '<testcase name="nginx_1.19.0" classname="CVE-2024-1234">'

    # Second: GHSA-abcd-1234 as classname, golang_1.20.0 as name
    assert_output --partial '<testcase name="golang_1.20.0" classname="GHSA-abcd-1234">'

    # Third: no underscore, use full ruleId as classname, "unknown" as name
    assert_output --partial '<testcase name="unknown" classname="no-underscore-format">'
}

@test "sanitizes XML-special characters" {
    local sarif_file="${BATS_TEST_TMPDIR}/special-chars.sarif"
    cat > "${sarif_file}" <<'EOF'
{
  "version": "2.1.0",
  "runs": [{
    "tool": {"driver": {"name": "roxctl"}},
    "results": [
      {
        "ruleId": "CVE-2024-1234_lib<\"test\">&apos_1.0",
        "level": "error",
        "message": {"text": "vuln with <special> & \"chars\""}
      }
    ]
  }]
}
EOF

    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh" "${sarif_file}" "image&<>name"
    assert_success

    local junit_file="${ARTIFACT_DIR}/junit-misc/junit-image&<>name.xml"
    assert [ -f "${junit_file}" ]

    run cat "${junit_file}"
    # XML-special chars in attributes are replaced with spaces
    refute_output --partial 'name="image&<>name"'
    assert_output --partial 'name="image   name"'
}

@test "fails on malformed sarif" {
    local sarif_file="${BATS_TEST_TMPDIR}/bad.sarif"
    echo "not json" > "${sarif_file}"

    run "${BATS_TEST_DIRNAME}/sarif-to-junit.sh" "${sarif_file}" "test-image:latest"
    assert_failure
    assert_output --partial 'Failed to parse SARIF'
}
