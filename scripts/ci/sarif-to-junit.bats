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

    # Verify success JUnit record was created
    local junit_file="${ARTIFACT_DIR}/junit-misc/junit-test-image:latest.xml"
    assert [ -f "${junit_file}" ]
    run grep '<testsuite' "${junit_file}"
    assert_success
    assert_output --partial 'failures="0"'
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

    # Verify JUnit files were created for each CVE
    local junit_file1="${ARTIFACT_DIR}/junit-misc/junit-CVE-2024-1234_nginx_1.19.0.xml"
    local junit_file2="${ARTIFACT_DIR}/junit-misc/junit-CVE-2024-5678_openssl_1.1.1.xml"

    assert [ -f "${junit_file1}" ]
    assert [ -f "${junit_file2}" ]

    # Verify first failure contains expected details
    run cat "${junit_file1}"
    assert_output --partial 'CVE-2024-1234'
    assert_output --partial 'Severity: error'
    assert_output --partial 'Critical vulnerability in nginx'
    assert_output --partial 'failures="1"'

    # Verify second failure
    run cat "${junit_file2}"
    assert_output --partial 'CVE-2024-5678'
    assert_output --partial 'Severity: warning'
    assert_output --partial 'Important vulnerability in openssl'
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
    local junit_file="${ARTIFACT_DIR}/junit-misc/junit-CVE-2024-9999_test_1.0.0.xml"
    assert [ -f "${junit_file}" ]
}
