# StackRox CI Test Artifacts - Complete Failure Triage Guide

This directory contains test artifacts from StackRox CI pipeline failures. This comprehensive guide provides everything needed to efficiently triage CI failures, from initial investigation to root cause analysis.

## 🚀 Quick Start Triage Workflow

### Step 1: JIRA Investigation (1 minute) - START HERE FIRST
1. **Use MCP JIRA tools** to get full issue details: `mcp__mcp-atlassian__jira_get_issue` with the ROX-XXXXX ticket
2. **Check latest comments** - Recent failure info is usually in the most recent comment, not the original description
3. **Search for related issues** using JQL queries to identify patterns or duplicates
4. **Identify failure type**: Known flaky test vs new failure vs recurring infrastructure issue

### Step 2: Initial Artifact Assessment (30 seconds)
1. **Check overall job status**: `cat finished.json | grep '"result"'` (look for `"FAILURE"`)
2. **Identify test platform**: Look for directory names (`aks-qa-e2e-tests`, `osd-aws-qa-e2e-tests`, etc.)
3. **Check for known issues**: Open `junit2jira-summary.html` to see if this is a known flaky test

### Step 3: Download Complete Artifacts (Recommended for thorough analysis)
#### From JIRA Issue:
1. **Check JIRA comments first** - Latest failure info is usually in the most recent comment, not the original description
2. **Find latest Build ID** - Look in the newest comment for the most recent build ID (e.g., `1963388448995807232`)
3. **Click Build ID link** to navigate to Prow page - this shows the build status and artifacts location
4. **Find correct bucket path** - Prow page reveals whether artifacts are in `test-platform-results` or `origin-ci-test`
5. **Use gsutil to download**: `gsutil -m cp -r "gs://test-platform-results/logs/{job-name}/{build-id}/*" ./local-artifacts/`

**⚠️ Important**: JIRA issues are automatically updated with new failures. Always check comments for the latest build information, not just the original issue description.

#### From Artifact Summary:
1. **Navigate to**: `artifacts/<platform>-e2e-tests/stackrox-stackrox-e2e-test/artifacts/howto-locate-other-artifacts-summary.html`
2. **Copy the `gsutil` command** from this file to download all artifacts locally
3. **Benefits**: Enables offline analysis, grep across all files, better log viewing tools

#### Common Patterns:
- **Standard tests**: `gs://test-platform-results/logs/{job-name}/{build-id}/`
- **Some PR tests**: `gs://origin-ci-test/pr-logs/pull/stackrox_stackrox/{pr-number}/{job-name}/{build-id}/`
- **Merge tests**: `gs://origin-ci-test/logs/{job-name}/{build-id}/`

### Step 4: Locate Primary Failure (1-2 minutes)
1. **Navigate to main test step**: `stackrox-stackrox-e2e-test/artifacts/`
2. **Quick scan of failures**: Check `junit-misc/` for infrastructure failures
3. **Review test execution**: Look at `spec-logs-summary.html` for test-specific failures

### Step 5: Deep Dive Analysis (5-10 minutes)
1. **Examine specific test logs**: Use individual `*Test.log` files in `spec-logs/`
2. **Check environment**: Review `debug.txt` for configuration issues
3. **Correlate with source code**: Match failing tests to source files in `qa-tests-backend/src/test/groovy/`

### 💡 Pro Tips for Efficient Triage
- **Start with `junit2jira-summary.html`**: Saves time if it's a known flaky test
- **Use local downloads**: For complex issues, download artifacts locally for better search capabilities
- **Check multiple platforms**: If available, compare failure across different test platforms
- **Focus on infrastructure first**: Most failures are environment-related, not test logic issues

## Directory Structure Overview

The test artifacts are organized by build ID (numeric directories like `1943468082932486144/`), each containing:

### Top-Level Files
- **`prowjob.json`** - Prow job configuration and metadata
- **`finished.json`** - Job completion status and result (`"result":"FAILURE"` indicates failure)
- **`build-log.txt`** - High-level build execution log
- **`started.json`** - Job start timestamp and metadata
- **`prowjob_junit.xml`** - JUnit results for the overall job
- **`sidecar-logs.json`** - Sidecar container logs

### Core Artifacts Directory (`artifacts/`)

#### Build Information
- **`metadata.json`** - Contains repo, commit SHA, and workspace info
- **`ci-operator.log`** - Detailed CI operator execution log
- **`ci-operator-step-graph.json`** - Visual representation of build steps
- **`ci-operator-metrics.json`** - Performance metrics
- **`junit_operator.xml`** - CI operator JUnit results

#### Build Logs and Resources
- **`build-logs/`** - Contains logs for specific build targets:
  - `src-amd64.log` - Source build logs
  - `test-bin-amd64.log` - Test binary build logs
- **`build-resources/`** - Kubernetes resources created during build:
  - `builds.json`, `events.json`, `imagestreams.json`, `pods.json`, `templateinstances.json`

## Test Suite Artifacts

### Platform-Specific Test Suites
Different builds test on various platforms:
- **`aks-qa-e2e-tests/`** - Azure Kubernetes Service tests
- **`osd-aws-qa-e2e-tests/`** - OpenShift Dedicated on AWS tests
- **`aro-qa-e2e-tests/`** - Azure Red Hat OpenShift tests
- **`merge-qa-e2e-tests/`** - Merge queue tests (OpenShift 4.x)

### Test Lifecycle Steps
Each test suite contains standard lifecycle steps:

1. **`*-create/`** - Cluster creation phase
2. **`stackrox-stackrox-begin/`** - StackRox deployment initialization
3. **`stackrox-stackrox-e2e-test/`** - Main E2E test execution
4. **`stackrox-stackrox-end/`** - Cleanup and teardown
5. **`*-destroy/`** - Cluster destruction

### Key Failure Analysis Files

#### Essential Files for Each Step
- **`build-log.txt`** - Step execution log
- **`finished.json`** - Step completion status
- **`artifacts/debug.txt`** - Environment variables and debug info
- **`sidecar-logs.json`** - Container sidecar logs

#### E2E Test Artifacts (`stackrox-stackrox-e2e-test/artifacts/`)

**Quick Status Files:**
- **`junit2jira-summary.html`** - Known flaky tests and JIRA links (example: ROX-30083 ImageSignatureVerificationTest failure)
- **`cluster-version.html`** - Target cluster version information
- **`howto-locate-other-artifacts-summary.html`** - Guide to artifact locations

**JUnit Results:**
- **`junit-misc/`** - Infrastructure and health check results:
  - `junit-Check unexpected pod restarts.xml` - Pod restart monitoring
  - `junit-Image_Availability.xml` - Image availability checks
  - `junit-OOM Check.xml` - Out of memory detection
  - `junit-Stackrox_Deployment.xml` - StackRox deployment status
  - `junit-SuspiciousLog-*.xml` - Log analysis for each component
  - `junit-image-prefetcher-*.xml` - Image prefetching results

**Detailed Test Logs:**
- **`spec-logs/`** - Individual test suite logs:
  - `*Test.log` files for each test class (e.g., `AuthServiceTest.log`, `NetworkFlowTest.log`)
  - `global.log` - Overall test execution log
  - `spec-logs-summary.html` - Test results summary

**Other Artifacts:**
- **`reports/`** - Test reports and output files
- **`junit-part-1-tests/`** - First part of test results
- **`webhook_server_port_forward.log`** - Webhook server connectivity logs

## Quick Triage Steps

### 1. Check Job Status
```bash
# Check if job failed
cat finished.json | grep '"result"'
# Look for: "result":"FAILURE"
```

### 2. Identify Failed Step
```bash
# Check prowjob for overall context
cat prowjob.json | grep '"context"'
# Check each test step's finished.json for failures
```

### 3. Review Known Issues
- Check `junit2jira-summary.html` for known flaky tests
- Look for existing JIRA tickets (ROX-XXXXX format)

### 4. Examine Test Failures
- Review `junit-misc/` for infrastructure issues
- Check specific test logs in `spec-logs/` for detailed failure info
- Look for patterns in `junit-SuspiciousLog-*.xml` files

### 5. Check Infrastructure
- Review `junit-Check unexpected pod restarts.xml` for stability issues
- Examine `junit-OOM Check.xml` for memory problems
- Check `debug.txt` files for environment issues

## 🔍 Common Infrastructure Failure Patterns

### Pod and Container Issues
- **Pod Restarts**: Check `junit-Check unexpected pod restarts.xml` for `failures="0"` (good) vs `failures="1+"` (problematic)
- **Image Problems**: Review `junit-Image_Availability.xml` for registry connectivity or authentication issues
- **Resource Constraints**: Examine `junit-OOM Check.xml` for memory limits and resource quota problems
- **Deployment Status**: Verify `junit-Stackrox_Deployment.xml` for StackRox component readiness

### Environment and Configuration
- **Debug Information**: Always check `debug.txt` files for environment variable misconfigurations
- **Build Logs**: Review `build-log.txt` at step level for detailed execution information
- **Webhook Connectivity**: Check `webhook_server_port_forward.log` for admission controller issues

### Known Flaky Tests (Check First!)
- **JIRA Integration**: Always review `junit2jira-summary.html` first for known issues
- **Example Pattern**: "ROX-30083: ImageSignatureVerificationTest / initializationError FAILED"
- **Retry Strategy**: Many flaky tests are automatically retried via `@OnFailure` annotations in source code

### Timing and Resource Issues
- **Cluster Creation**: Check `*-create/finished.json` for cloud provider timeouts or quota limits
- **Test Execution**: Look for "timeout" messages in individual test logs
- **Cleanup Problems**: Review `*-destroy/` or `stackrox-stackrox-end/` for resource cleanup issues

## StackRox Test Framework Details

### Test Source Code Structure
Based on the StackRox repository at `/tmp/tests/stackrox/`, the test framework consists of:

#### QA Test Backend (`qa-tests-backend/`)
- **Language**: Groovy with Spock framework
- **Location**: `qa-tests-backend/src/test/groovy/`
- **Base Class**: `BaseSpecification.groovy` - Provides common test infrastructure, setup/teardown
- **Test Discovery**: Check `spec-logs-summary.html` for complete list of test classes and their logs
- **Naming Pattern**: `{TestName}Test.groovy` or `{TestName}.groovy` files
- **Coverage**: Main QA test suite covering authentication, policies, scanning, network, compliance, deployments, integrations, etc.

#### Go-Based Tests (`tests/`)
- **Language**: Go with testing framework (not Groovy)
- **Location**: `tests/` directory (separate from `qa-tests-backend/`)
- **Test Types**: Compliance, compatibility, and specialized integration tests
- **Key Go Test Files**:
  - `compliance_operator_v2_test.go` - Compliance Operator v2 integration tests
  - `pods_test.go` - Pod creation and management tests
  - `centralgrpc.go` - gRPC connection utilities for Central
  - `common.go` - Common test utilities and helpers

#### Test Framework Distinction
**IMPORTANT**: StackRox has two separate test frameworks:

1. **Groovy/Spock Tests** (`qa-tests-backend/src/test/groovy/`):
   - Main QA test suite for most StackRox functionality
   - Uses Spock framework with Groovy
   - Artifact pattern: `spec-logs/{TestName}.log`
   - JUnit results in: `junit-part-1-tests/`, `junit-part-2-tests/`

2. **Go Tests** (`tests/`):
   - Specialized tests for compliance, compatibility, and specific integrations
   - Native Go testing framework
   - Artifact pattern: `junit-{test-type}-results/{test-type}-results/`
   - Examples: `junit-compliance-v2-tests-results/`, `junit-compatibility-test-*/`

#### Job Type Mapping
- **Standard QA Tests**: `*-qa-e2e-tests` jobs → Groovy tests in `qa-tests-backend/`
- **Compliance Tests**: `*-compliance-e2e-tests` jobs → Go tests in `tests/`
- **Compatibility Tests**: `*-compatibility-tests` jobs → Go tests in `tests/`
- **NonGroovy Tests**: `*-nongroovy-*` jobs → Go tests in `tests/`

### Important Artifact Files

#### Test Results Summary Files
- **`spec-logs-summary.html`** - Auto-generated HTML summary linking to all test class logs
  - Created by `qa-tests-backend/scripts/lib.sh:surface_spec_logs()`
  - Provides clickable links to individual test logs with OpenShift CI URLs
  - Essential for quick navigation to specific test failures

#### Artifact Fetching Guide
- **`howto-locate-other-artifacts-summary.html`** - Contains `gsutil` commands for downloading complete artifact sets
  - Path: `artifacts/<platform>-e2e-tests/stackrox-stackrox-e2e-test/artifacts/howto-locate-other-artifacts-summary.html`
  - Purpose: Provides Google Cloud Storage commands to download all test artifacts locally
  - Usage: Copy the `gsutil` command from this file to download complete test runs for local analysis

### Test Artifact URLs and Access

#### OpenShift CI Artifact URLs
Test logs are accessible via structured URLs, but **IMPORTANT**: Always check the Prow build page from JIRA to get the correct bucket location.

**Web Access** (view only):
- **PR builds**: `https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/pr-logs/pull/stackrox_stackrox/{PR_NUMBER}/{JOB_NAME}/{BUILD_ID}/artifacts/`
- **Merge builds**: `https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/logs/{JOB_NAME}/{BUILD_ID}/artifacts/`

#### Google Cloud Storage Integration
- **Storage Backend**: Multiple GCS buckets (`test-platform-results`, `origin-ci-test`)
- **Access**: `test-platform-results` is generally accessible, `origin-ci-test` may have restrictions
- **Best Practice**: Always use Prow build page from JIRA to find correct bucket and path
- **Download Tool**: Use `gsutil` commands - get exact paths from Prow page or `howto-locate-other-artifacts-summary.html`
- **Retention**: Artifacts are retained for historical analysis and debugging

### Advanced Debugging

#### Cross-referencing Logs to Source Code
When analyzing failures, match log files and JUnit results to source code:

**Groovy Tests** (Standard QA):
- **Pattern**: `{TestName}.log` → `qa-tests-backend/src/test/groovy/{TestName}.groovy`
- **Discovery**: Use `spec-logs-summary.html` to see all test logs and their corresponding source files
- **JUnit Results**: `junit-part-1-tests/`, `junit-part-2-tests/`
- **Examples**: `AuthServiceTest.log` → `AuthServiceTest.groovy`, `NetworkFlowTest.log` → `NetworkFlowTest.groovy`

**Go Tests** (Compliance/Compatibility/NonGroovy):
- `junit-compliance-v2-tests-results/report.xml` → `tests/compliance_operator_v2_test.go`
- `junit-compatibility-test-*/report.xml` → `tests/pods_test.go`, `tests/common.go`
- Test logs in: `junit-{test-type}-results/{test-type}-results/test.log`
- Error traces show exact file:line locations (e.g., `/go/src/github.com/stackrox/stackrox/tests/compliance_operator_v2_test.go:211`)

#### Test Infrastructure Details
**Groovy Tests** (`qa-tests-backend/`):
- **Framework**: Groovy with Spock testing framework
- **Retry Logic**: Tests include retry mechanisms via `@OnFailure` annotations
- **Debug Collection**: Automatic debug artifact collection on test failures

**Go Tests** (`tests/`):
- **Framework**: Native Go testing with `testing` package
- **Utilities**: Shared utilities in `common.go`, `centralgrpc.go`
- **Error Handling**: Structured error reporting with precise file:line references

**Common Infrastructure**:
- **Environment Integration**: Both test types deploy to real Kubernetes clusters (AKS, OSD, ARO, OCP)
- **CI Integration**: Both integrate with OpenShift CI and JIRA for automated issue creation

## Artifact File Creation and Content Details

### JUnit XML Files (`junit-misc/`)
**Source**: Created by `scripts/ci/lib.sh` functions
- **Creation Functions**:
  - `save_junit_success()` - Records passing tests
  - `save_junit_failure()` - Records failing tests with error details
  - `save_junit_skipped()` - Records skipped tests
- **File Location**: `${ARTIFACT_DIR}/junit-misc/junit-{CLASS}.xml`
- **Content Structure**:
  - Test suite name, total tests, failures, skipped counts
  - Individual test cases with names, status, and failure messages
  - Failure details are base64-encoded to handle multiline output
- **Examples from Repository**:
  - `junit-Check unexpected pod restarts.xml` - Pod restart monitoring results
  - `junit-Image_Availability.xml` - Image availability validation
  - `junit-OOM Check.xml` - Out of memory detection results
  - `junit-Stackrox_Deployment.xml` - StackRox deployment status
  - `junit-SuspiciousLog-*.xml` - Log analysis for each component
  - `junit-Cluster.xml` - Cluster creation/destruction status (generated by `.openshift-ci/end.sh`)

### JUnit Printer Framework (`pkg/printers/junit.go`)
**Purpose**: Converts JSON data to JUnit XML format for roxctl commands
- **Usage**: Used by `roxctl image check` and `roxctl deployment check` commands
- **Input**: JSON objects with GJSON path expressions
- **Output**: Standard JUnit XML with test suites and test cases
- **Key Features**:
  - Supports failed, skipped, and successful test cases
  - Uses GJSON expressions to extract test data from JSON
  - XML escaping for test names and messages

### Debug Files (`debug.txt`)
**Source**: Generated during CI step execution
- **Location**: `artifacts/{step-name}/artifacts/debug.txt`
- **Content**: Environment variable dump showing:
  - CI environment variables (BUILD_ID, JOB_NAME, etc.)
  - Kubernetes configuration (KUBECONFIG paths)
  - StackRox-specific variables (ROX_*, CLUSTER_*, etc.)
  - Test execution parameters
- **Creation**: Automatic dump during CI step initialization

### Spec Logs (`spec-logs/`)
**Source**: Groovy/Spock test execution output
- **Individual Test Logs**: One `.log` file per test class (e.g., `AuthServiceTest.log`)
- **Content**: Detailed test execution logs including:
  - Test setup and teardown operations
  - API calls and responses
  - Kubernetes operations
  - Error messages and stack traces
  - Debug output from test methods
- **Global Log**: `global.log` contains overall test execution information

### HTML Summary Files

#### `spec-logs-summary.html`
**Source**: Generated by `qa-tests-backend/scripts/lib.sh:surface_spec_logs()`
- **Purpose**: Provides clickable links to individual test class logs
- **Content**: HTML page with list of test classes linking to OpenShift CI URLs
- **URL Structure**:
  - PR builds: `https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/pr-logs/pull/stackrox_stackrox/{PR_NUMBER}/{JOB_NAME}/{BUILD_ID}/artifacts/`
  - Merge builds: `https://gcsweb-ci.apps.ci.l2s4.p1.openshiftapps.com/gcs/origin-ci-test/logs/{JOB_NAME}/{BUILD_ID}/artifacts/`

#### `junit2jira-summary.html`
**Source**: Generated by `junit2jira` GitHub Action (`.github/actions/junit2jira/`)
- **Purpose**: Links JUnit test failures to existing JIRA tickets
- **Content**: HTML page showing "Possible Flake Tests" with links like:
  - "ROX-30083: ImageSignatureVerificationTest / initializationError FAILED"
- **Integration**: Automatically creates JIRA tickets for new test failures
- **Tool**: External `junit2jira` binary downloaded from GitHub releases

#### `howto-locate-other-artifacts-summary.html`
**Source**: Generated by `scripts/ci/store-artifacts.sh`
- **Purpose**: Provides `gsutil` commands for downloading complete artifact sets
- **Content**: Google Cloud Storage download instructions
- **Usage**: Copy the `gsutil` command to download all test artifacts locally
- **Storage Backend**: `gs://stackrox-ci-artifacts` bucket

### Build and Infrastructure Logs

#### `build-log.txt`
**Source**: CI step execution output
- **Content**: Complete build/test execution log for each step
- **Location**: Present at multiple levels (top-level and per-step)

#### `ci-operator.log`
**Source**: OpenShift CI operator execution
- **Content**: Detailed CI pipeline execution including:
  - Image building and tagging
  - Step orchestration
  - Resource creation and management
  - Error handling and debugging information

#### `webhook_server_port_forward.log`
**Source**: Generated by `scripts/ci/create-webhookserver.sh`
- **Purpose**: Webhook server connectivity logs
- **Content**: Port forwarding logs for webhook server testing
- **Creation**: Automatic background process during webhook server deployment

### Metadata and Configuration Files

#### `metadata.json`
**Content**: Repository and build metadata including:
- Repository information (`"repo": "stackrox/stackrox"`)
- Git commit SHA (`"repos": {"stackrox/stackrox": "branch:commit-sha"}`)
- CI workspace details (`"work-namespace": "ci-op-xxxxx"`)
- Pod and job identifiers

#### `finished.json`
**Content**: Step completion status
- Result status (`"result": "FAILURE"` or `"result": "SUCCESS"`)
- Timestamp information
- Metadata about the completed step

#### `ci-operator-step-graph.json` and `ci-operator-metrics.json`
**Purpose**: CI pipeline visualization and performance data
- Step dependencies and execution order
- Resource usage and timing metrics
- Pipeline optimization information

### Integration with External Tools

#### JIRA Integration
- **Tool**: `junit2jira` GitHub Action
- **Purpose**: Automatically creates JIRA tickets for test failures
- **Input**: JUnit XML files from `junit-misc/` directory
- **Output**: `junit2jira-summary.html` with links to existing tickets
- **Configuration**: Uses JIRA_TOKEN secret for authentication

#### Google Cloud Storage
- **Purpose**: Long-term artifact storage and sharing
- **Tools**: `gsutil` commands in `store-artifacts.sh`
- **Access**: Via `howto-locate-other-artifacts-summary.html` instructions
- **Retention**: Historical artifact access for debugging and analysis

## 📥 Downloaded Artifacts Deep Dive

### Overview of Downloaded Content
When you download artifacts using the `gsutil` command from `howto-locate-other-artifacts-summary.html`, you get access to much richer debugging information than what's available in the basic OpenShift CI artifacts.

### 🔥 Most Valuable Files for Debugging

#### 1. StackRox Namespace Pod Logs 📋
**Location**: `final/qa-tests-backend-logs/{UUID}/stackrox-k8s-logs/stackrox/pods/`

**Essential Pod Logs to Check:**
- **`central-{hash}-central.log`** - StackRox Central service logs
  - Contains: Database migrations, API calls, policy evaluations, authentication events
  - **Search for**: `ERROR`, `WARN`, `Failed`, `timeout`, `certificate`, `database`
  - **Key patterns**: Migration failures, DB connection issues, certificate problems

- **`scanner-{hash}-scanner.log`** - Scanner service logs
  - Contains: Image scanning progress, vulnerability detection, scanner startup
  - **Search for**: `error`, `Failed to open database`, `connection refused`, `timeout`
  - **Common issues**: Scanner DB connectivity, definition loading failures

- **`sensor-{hash}-sensor.log`** - Sensor service logs
  - Contains: Policy enforcement, admission control decisions, cluster monitoring
  - **Search for**: `ERROR`, `admission`, `policy`, `webhook`, `denied`

- **`admission-control-{hash}-admission-control.log`** - Admission controller logs
  - Contains: Webhook validation decisions, policy enforcement
  - **Search for**: `denied`, `failed`, `webhook`, `timeout`, `certificate`

- **`collector-{hash}-collector.log`** & `collector-{hash}-compliance.log`** - Collector logs
  - Contains: Runtime monitoring, process tracking, compliance checks
  - **Search for**: `error`, `failed`, `runtime`, `process`, `compliance`

#### 2. Kubernetes Events 📅
**Location**: `final/qa-tests-backend-logs/{UUID}/stackrox-k8s-logs/stackrox/events.txt`

**Critical Event Types:**
- **`Warning` events** - Usually indicate problems
- **`FailedToRetrieveImagePullSecret`** - Image registry authentication issues
- **`Killing/Stopped`** - Pod restarts or terminations
- **`FailedMount`** - Volume mounting issues
- **`BackOff`** - Container restart loops

#### 3. Database and State Information 💾
**Location**: `final/qa-tests-backend-logs/{UUID}/central-data/`

**Key Files:**
- **`postgres_db_{timestamp}.sql.zip`** - Complete database dump
  - Contains: All policies, configurations, scan results at test completion
  - **Use for**: Understanding what state the system was in when tests failed

- **`policies.json`** - All security policies active during test
  - Contains: Policy definitions, rules, lifecycle stages
  - **Search for**: Policy names mentioned in failing tests

- **`imageintegrations.json`** - Image registry configurations
  - Contains: Registry credentials, authentication settings
  - **Use for**: Debugging image scanning or pulling issues

#### 4. Performance Metrics 📊
**Location**: `part-1/collector-metrics/`

**Files**: `collector-{hash}.txt`
- **Contains**: Prometheus metrics from collector pods
- **Search for**: Memory usage, event counts, error rates
- **Key metrics**: `rox_collector_events`, memory/CPU usage patterns

#### 5. Debug and Diagnostic Bundles 🔧
**Location**: `part-1/debug-dump/` and `part-1/diagnostic-bundle/`

**Files**:
- `stackrox_debug_{timestamp}.zip` - Complete system state dump
- `stackrox_diagnostic_{timestamp}.zip` - Diagnostic information

**Contents**: Central logs, database state, configuration, system info

#### 6. Image Scan Results 🔍
**Location**: `final/qa-tests-backend-logs/{UUID}/image-scans/`

**Files**: `*.json` - Detailed vulnerability scan results
- **Use for**: Understanding why image scanning tests failed
- **Contains**: CVE details, package vulnerabilities, scan metadata

### 🔍 Key Search Patterns

#### For Infrastructure Issues:
```bash
# Search across all logs for critical errors
grep -r "ERROR\|FATAL\|WARN" stackrox-k8s-logs/
grep -r "connection refused\|timeout\|failed" stackrox-k8s-logs/
grep -r "certificate\|tls\|ssl" stackrox-k8s-logs/
```

#### For Authentication/Authorization Issues:
```bash
# Look for auth-related problems
grep -r "authentication\|authorization\|denied\|forbidden" stackrox-k8s-logs/
grep -r "certificate\|expired\|invalid" stackrox-k8s-logs/
```

#### For Database Issues:
```bash
# Database connectivity and migration problems
grep -r "database\|postgres\|migration\|sql" stackrox-k8s-logs/
grep -r "connection refused.*5432\|dial tcp.*5432" stackrox-k8s-logs/
```

#### For Scanner Issues:
```bash
# Scanner service problems
grep -r "scanner\|vulnerability\|scan.*failed" stackrox-k8s-logs/
grep -r "definitions.*failed\|repo-to-cpe" stackrox-k8s-logs/
```

### 🔗 Correlating Test Failures with Pod Logs

#### Finding Specific Test Errors in Pod Logs
When a test fails, you need to correlate the test execution time with pod log entries:

1. **Get Test Timestamps**:
   - Check `spec-logs/{TestName}.log` for test execution times
   - Note the exact time when the test failed (timestamps in logs)
   - Example: `2025-07-11 01:15:23` from test log

2. **Search Pod Logs Around Test Time**:
```bash
# Find errors around specific test execution time
grep -A5 -B5 "2025-07-11 01:1[5-6]" stackrox-k8s-logs/stackrox/pods/central-*-central.log
grep -A5 -B5 "2025-07-11 01:1[5-6]" stackrox-k8s-logs/stackrox/pods/sensor-*-sensor.log
```

3. **Match Test Actions to Pod Responses**:
   - **API Test Failures** → Check `central-*-central.log` for API error responses
   - **Policy Test Failures** → Check `sensor-*-sensor.log` and `admission-control-*` logs
   - **Image Scanning Failures** → Check `scanner-*-scanner.log` for scan errors
   - **Network Test Failures** → Check `collector-*-collector.log` for network events

#### Test-Specific Log Correlation Patterns

**For Authentication Tests** (`AuthServiceTest.log`):
```bash
# Look for auth-related errors in central logs during test time
grep -C3 "authentication\|login\|token\|unauthorized" central-*-central.log
grep -C3 "certificate\|tls.*error\|ssl.*error" central-*-central.log
```

**For Image Scanning Tests** (`ImageScanningTest.log`):
```bash
# Correlate scanning failures with scanner pod logs
grep -C3 "scan.*failed\|scanner.*error\|vulnerability.*error" scanner-*-scanner.log
grep -C3 "database.*error\|connection.*refused.*5432" scanner-*-scanner.log
```

**For Policy Tests** (`PolicyConfigurationTest.log`):
```bash
# Check policy evaluation in sensor and admission-control logs
grep -C3 "policy.*violation\|admission.*denied\|webhook.*failed" sensor-*-sensor.log
grep -C3 "policy.*evaluation\|admission.*error" admission-control-*-admission-control.log
```

**For Network Tests** (`NetworkFlowTest.log`):
```bash
# Check network monitoring in collector logs
grep -C3 "network.*flow\|connection.*denied\|network.*policy" collector-*-collector.log
grep -C3 "network.*baseline\|flow.*detection" collector-*-compliance.log
```

**For Deployment Tests** (`DeploymentTest.log`):
```bash
# Check deployment monitoring across multiple pods
grep -C3 "deployment.*create\|deployment.*delete" sensor-*-sensor.log
grep -C3 "resource.*creation\|namespace.*event" central-*-central.log
```

#### Advanced Correlation Techniques

1. **Find API Calls Made by Tests**:
```bash
# Search for specific API endpoints that tests hit
grep -C2 "POST\|GET\|PUT\|DELETE.*api" central-*-central.log
grep -C2 "graphql\|grpc" central-*-central.log
```

2. **Track Object Creation/Deletion**:
```bash
# Find when test objects were created/deleted
grep -C3 "Created\|Deleted.*deployment\|namespace\|policy" sensor-*-sensor.log
```

3. **Monitor Resource Changes**:
```bash
# Check for resource constraint issues during heavy tests
grep -C3 "memory\|cpu\|resource.*limit\|throttl" collector-*-collector.log
```

#### Using Events to Pinpoint Issues
The `events.txt` file is crucial for timeline correlation:
```bash
# Find events around test failure time
grep "2025-07-11 01:1[5-6]" events.txt
# Look for Warning/Error events near test execution
awk '/Warning|Error/ && /2025-07-11 01:1[5-6]/' events.txt
```

### 🔍 Tracing Failures Back to Source Code

#### Getting the Exact Code Version
To understand what code was actually running during the test failure:

1. **Find the Exact Commit SHA**:
   - Check `metadata.json` for the commit that was tested
   - Look in `build-log.txt` for version information
   - Example: `"repos": {"stackrox/stackrox": "main:a1b2c3d4e5f6..."}`

2. **Clone and Checkout the Repository**:
```bash
# Clone the StackRox repository
git clone https://github.com/stackrox/stackrox.git
cd stackrox

# Checkout to the exact commit that was tested
git checkout a1b2c3d4e5f6  # Use SHA from metadata.json/build-log.txt
```

3. **Verify You Have the Right Version**:
```bash
# Confirm you're on the right commit
git log --oneline -1
git show --name-only  # See what files changed in this commit
```

#### Mapping Test Failures to Source Code

**Understanding the Separation**:
- **Integration tests** are in `qa-tests-backend/src/test/groovy/` (Groovy/Spock)
- **System under test** is the StackRox platform code (Go services)
- When tests fail, the issue is usually in the **Go services**, not the test code

**Finding Error Origins**:
1. **Extract Exact Error Messages** from pod logs:
   ```bash
   # From downloaded artifacts, get specific error messages
   grep -A3 -B3 "ERROR\|FATAL" stackrox-k8s-logs/stackrox/pods/central-*-central.log
   ```

2. **Search Source Code for Error Messages**:
   ```bash
   # In the cloned repository, search for the exact error strings
   grep -r "Failed to connect to database" . --include="*.go"
   grep -r "scanner.*error\|vulnerability.*failed" . --include="*.go"
   grep -r "admission.*denied\|policy.*violation" . --include="*.go"
   ```

#### Service-Specific Code Investigation

**Central Service Errors** (`central-*-central.log`):
- **Code Location**: `central/` directory
- **Key Components**: Database migrations, API handlers, authentication
```bash
# Search for database/migration related errors
grep -r "Migration.*failed\|DB.*connection" central/ migrator/
grep -r "postgres\|database.*error" pkg/postgres/
```

**Scanner Service Errors** (`scanner-*-scanner.log`):
- **Code Location**: `scanner/` directory
- **Key Components**: Vulnerability scanning, definition loading
```bash
# Search for scanner-specific errors
grep -r "Failed to open database.*scanner" scanner/
grep -r "Loading.*definitions.*failed\|repo-to-cpe" scanner/
```

**Sensor Service Errors** (`sensor-*-sensor.log`):
- **Code Location**: `sensor/` directory
- **Key Components**: Policy enforcement, resource monitoring
```bash
# Search for policy enforcement errors
grep -r "policy.*evaluation\|admission.*webhook" sensor/
grep -r "kubernetes.*api.*error" sensor/
```

**Admission Controller Errors** (`admission-control-*-admission-control.log`):
- **Code Location**: `pkg/admission/` and related webhook code
```bash
# Search for webhook and validation errors
grep -r "webhook.*timeout\|certificate.*validation" pkg/admission/
grep -r "admission.*denied" pkg/admission/
```

#### Advanced Code Analysis

**Finding Recent Changes That Might Cause Issues**:
```bash
# See what changed in the last few commits
git log --oneline -10
git show HEAD --name-only

# Check specific files that changed recently
git log -p --follow central/auth.go  # See recent changes to auth code
git blame central/database.go        # See who changed database code when
```

**API Endpoint to Code Mapping**:
```bash
# When tests hit specific endpoints, find the handler code
grep -r "api/v1/policies" central/    # Policy management APIs
grep -r "api/v1/images" central/      # Image scanning APIs
grep -r "api/v1/deployments" central/ # Deployment APIs
grep -r "graphql" central/             # GraphQL resolvers
```

**Configuration and Environment Code**:
```bash
# Find code that handles environment variables from debug.txt
grep -r "ROX_.*\|STACKROX_" . --include="*.go"
grep -r "viper\|os.Getenv" . --include="*.go"
```

#### Test-to-Code Correlation Examples

**When `ImageScanningTest.groovy` fails**:
1. Get error from `scanner-*-scanner.log`: "Failed to connect to scanner database"
2. Search source: `grep -r "Failed to connect to scanner database" scanner/`
3. Check recent changes: `git log --oneline scanner/database/`
4. Analyze the failing code section and recent modifications

**When `PolicyConfigurationTest.groovy` fails**:
1. Get error from `sensor-*-sensor.log`: "Policy evaluation failed"
2. Search source: `grep -r "Policy evaluation failed" sensor/ pkg/compliance/`
3. Check policy engine changes: `git log --oneline pkg/compliance/`
4. Look for recent policy rule modifications

**When `AuthServiceTest.groovy` fails**:
1. Get error from `central-*-central.log`: "Certificate validation failed"
2. Search source: `grep -r "Certificate validation failed" central/ pkg/mtls/`
3. Check auth changes: `git log --oneline central/auth/ pkg/auth/`
4. Verify TLS/certificate handling code

### 🎯 Complete Failure Investigation Workflow

1. **Start with Test Logs**: Identify exact failure time and test actions from `spec-logs/{TestName}.log`
2. **Check Events Timeline**: Look at `events.txt` for Warning/Error events around test time
3. **Correlate Pod Logs**: Search relevant pod logs using test timestamps
4. **Extract Error Messages**: Get exact error strings from pod logs
5. **Get Source Code**: Clone https://github.com/stackrox/stackrox and checkout to commit from `metadata.json`
6. **Map Errors to Source**: Search the checked-out code for error message origins
7. **Analyze Recent Changes**: Use `git log` and `git show` to understand what changed recently
8. **Identify Root Cause**: Match the failing code location with recent modifications

### 💡 Pro Tips for Downloaded Artifacts

1. **Use timestamps**: Correlate events across different logs using timestamps
2. **Follow pod lifecycle**: Track pod creation → running → termination in events
3. **Check resource limits**: Look for OOM kills or resource constraint messages
4. **Verify networking**: Check for DNS resolution and service connectivity issues
5. **Compare configurations**: Use central-data files to understand test environment setup

## 🚨 Most Common CI Failure Types

Based on real-world StackRox CI experience, the majority of failures fall into these categories:

### 1. Network Issues 🌐

**Symptoms to Look For:**
- `connection refused`, `dial tcp`, `no route to host` in pod logs
- DNS resolution failures: `no such host`, `dns lookup failed`
- Service connectivity timeouts between StackRox components

**Quick Investigation:**
```bash
# Check for network-related errors across all pods
grep -r "connection refused\|dial tcp\|no route to host" stackrox-k8s-logs/stackrox/pods/
grep -r "dns.*failed\|no such host" stackrox-k8s-logs/stackrox/pods/
```

**Common Patterns:**
- **Central ↔ Database**: `connection refused.*5432` in central logs
- **Scanner ↔ Scanner DB**: `dial tcp.*scanner-db.*5432` in scanner logs
- **Inter-service communication**: `grpc.*connection.*failed` between services
- **External registry**: `dial tcp.*quay.io\|registry.*timeout` in events

**Where to Look:**
- `central-*-central.log` for database connection issues
- `scanner-*-scanner.log` for scanner database connectivity
- `events.txt` for DNS/networking events
- `collector-*-*.log` for node-level network issues

### 2. Timeout Issues ⏱️

**Symptoms to Look For:**
- `timeout`, `deadline exceeded`, `context deadline exceeded`
- `timed out waiting for`, `operation timeout`
- Tests hanging or taking much longer than usual

**Quick Investigation:**
```bash
# Find timeout-related errors
grep -r "timeout\|deadline exceeded\|timed out waiting" stackrox-k8s-logs/stackrox/pods/
grep -r "context.*deadline\|operation.*timeout" stackrox-k8s-logs/stackrox/pods/
```

**Common Timeout Types:**
- **Database Operations**: `postgres.*timeout`, `query.*timeout`
- **API Calls**: `grpc.*deadline exceeded`, `http.*timeout`
- **Image Scanning**: `scanner.*timeout`, `scan.*deadline`
- **Webhook Calls**: `admission.*timeout`, `webhook.*deadline`
- **Kubernetes API**: `k8s.*timeout`, `apiserver.*timeout`

**Performance Correlation:**
- Check `collector-metrics/` files for high resource usage during timeouts
- Look for memory pressure or CPU throttling patterns

### 3. Missing Resource Issues 📦

**Symptoms to Look For:**
- `not found`, `does not exist`, `missing`, `resource not available`
- Image pull failures, missing secrets, unavailable services
- Resource quota exceeded, insufficient permissions

**Quick Investigation:**
```bash
# Find missing resource errors
grep -r "not found\|does not exist\|missing" stackrox-k8s-logs/ events.txt
grep -r "resource.*not.*available\|insufficient" stackrox-k8s-logs/ events.txt
```

**Common Missing Resources:**
- **Images**: `image.*not found`, `pull.*failed` in events.txt
- **Secrets**: `secret.*not found`, `certificate.*missing`
- **ConfigMaps**: `configmap.*not found`
- **Services**: `service.*not found`, `endpoint.*not found`
- **Storage**: `pvc.*not found`, `volume.*not available`
- **RBAC**: `forbidden`, `access denied`, `insufficient privileges`

**Where to Focus:**
- `events.txt` - Most resource issues show up as Kubernetes events
- Pod `_describe.log` files - Show detailed resource status
- `secrets/` and `serviceaccounts/` subdirectories

### 🎯 Triage Approach for Common Issues

**All failure types (Network, Timeout, Missing Resources) are equally important** and should be investigated with the same urgency. The key difference is in investigation strategy:

#### Investigation Strategy by Type:

**Network Issues**:
- Start with `events.txt` to see if it's cluster-wide or service-specific
- Check database connectivity first (most critical for system function)
- Verify DNS resolution and service discovery

**Timeout Issues**:
- Check resource constraints in `collector-metrics/` first
- Look for memory/CPU pressure indicators
- Examine database query performance and API response times

**Missing Resource Issues**:
- Verify if resources were created initially or never existed
- Check RBAC permissions and quota limits
- Examine image availability and secret accessibility

#### Common Resolution Approaches:
- **Network/Timeout**: Often resolve with retry, but investigate underlying cause
- **Missing Resources**: Usually require manual intervention or infrastructure fixes
- **All Types**: Can indicate infrastructure instability requiring platform team involvement

### 🔍 Fast Network/Timeout/Resource Triage Commands

```bash
# One-liner to check for the most common issues
grep -r "connection refused\|timeout\|not found\|does not exist" stackrox-k8s-logs/stackrox/pods/ | head -20

# Check events for infrastructure issues
grep -E "Warning|Error" events.txt | grep -E "Failed|Timeout|NotFound"

# Quick resource availability check
ls stackrox-k8s-logs/stackrox/pods/ | grep -E "describe|object" | head -10
```

### 💡 Pro Tips for Common Failures

1. **Network Issues**: Check if problem is cluster-wide or service-specific
2. **Timeout Issues**: Look for resource constraints in collector metrics first
3. **Missing Resources**: Verify if resources existed before or never got created
4. **Retry Strategy**: Most network/timeout issues resolve on retry
5. **Infrastructure vs Code**: These failures are usually infrastructure, not code changes

## 👥 Team Ownership & Escalation

Based on the StackRox CODEOWNERS file, different teams own different components. When you identify the failing component, escalate to the appropriate team:

### **@stackrox/core-workflows Team**
**Responsible for**: Policies, detection, alerting, vulnerability management, database migrations

**When to Escalate**:
- Policy evaluation failures in sensor logs
- Database migration errors in central logs
- Vulnerability scanning issues (when not scanner service itself)
- Detection/alerting failures
- Search functionality issues

**Key Components**:
- `central/policy/`, `central/vulnmgmt/`, `central/reports/`
- `pkg/detection/`, `pkg/booleanpolicy/`, `pkg/postgres/`
- `migrator/` - Database migrations
- Default policies and policy management workflows

### **@stackrox/sensor-ecosystem Team**
**Responsible for**: Authentication, authorization, cloud sources, sensor, roxctl, telemetry

**When to Escalate**:
- Authentication/authorization failures
- Sensor connectivity issues
- Cloud source integration problems
- roxctl command failures
- SAC (Scoped Access Control) issues

**Key Components**:
- `sensor/` - Sensor service
- `roxctl/` - CLI tool
- `pkg/auth/`, `pkg/sac/` - Auth and access control
- Cloud sources and declarative config
- Image signatures

### **@stackrox/scanner Team**
**Responsible for**: Scanner service, image scanning, vulnerability definitions, registries

**When to Escalate**:
- Scanner service failures (`scanner-*-scanner.log` errors)
- Image integration issues
- Registry connectivity problems
- Scanner definition loading failures
- Image enrichment problems

**Key Components**:
- `scanner/` - Scanner service
- `central/imageintegration/`, `central/scannerdefinitions/`
- `pkg/registries/`, `pkg/scanners/`, `pkg/images/enricher/`
- Registry mirror configurations

### **@stackrox/ui Team**
**Responsible for**: User interface components

**When to Escalate**:
- UI-related test failures
- Frontend API interaction issues

**Key Components**:
- `ui/` directory

### **@stackrox/install Team**
**Responsible for**: Operator and installation components

**When to Escalate**:
- Deployment/installation failures
- Operator-related issues
- Helm chart problems

**Key Components**:
- `operator/` directory

### **QA Test Framework**: `@janisz`
**Responsible for**: Test framework itself (`qa-tests-backend/`)

**When to Escalate**:
- Test framework issues (not the tested functionality)
- Test infrastructure problems
- Groovy/Spock test execution issues

### 🎯 Quick Team Identification

**From Pod Log Errors**:
```bash
# Scanner team - if errors in scanner pods
grep -l "scanner" stackrox-k8s-logs/stackrox/pods/*.log

# Core team - if database/policy errors
grep -r "postgres\|policy.*evaluation\|migration" stackrox-k8s-logs/stackrox/pods/

# Sensor-ecosystem team - if auth/sensor errors
grep -r "authentication\|authorization\|sensor.*connection" stackrox-k8s-logs/stackrox/pods/
```

**From Source Code Changes**:
```bash
# Check which team owns the changed files
git show HEAD --name-only | head -5
# Then match against CODEOWNERS patterns above
```

**General Escalation Guidelines**:
- **Infrastructure issues** (network, timeout, resources): Platform/SRE team
- **Component-specific failures**: Use team mapping above
- **Test framework issues**: @janisz
- **Multiple component failures**: Start with @stackrox/core-workflows (central system)

## 🎫 Existing Tools & Dashboards Integration

### JIRA Integration
- **MCP JIRA Access**: Use the MCP JIRA integration tools to directly query and interact with JIRA issues during investigation
  - `mcp__mcp-atlassian__jira_get_issue` - Get detailed issue information
  - `mcp__mcp-atlassian__jira_search` - Search for related issues using JQL
  - `mcp__mcp-atlassian__jira_add_comment` - Add investigation findings to tickets
- **Automatic ticket creation**: The `junit2jira` tool automatically creates JIRA tickets for test failures
- **Continuous updates**: JIRA issues are automatically updated with new failures - each failure adds a comment with new build ID
- **Check existing tickets**: Always review `junit2jira-summary.html` first to see if the failure is already tracked
- **Latest failure location**: **Most recent failure information is in the latest JIRA comment**, not the original description
- **Build ID progression**: Comments show the failure history - use the newest build ID for investigation
- **Link CI failures to tickets**: Use JIRA ticket numbers (ROX-XXXXX) to track recurring issues
- **Escalation**: Reference existing JIRA tickets when escalating to teams

### Git Blame for Change Tracking
When you identify problematic code, use `git blame` to understand recent changes:

```bash
# After finding the failing code location
git blame path/to/failing/file.go | grep -A5 -B5 "error message location"

# See who changed specific lines recently
git log -p --follow path/to/file.go | head -50

# Find recent changes in a component
git log --oneline --since="1 week ago" central/auth/
```

**Using git blame effectively**:
1. Find the exact line causing the error from source code search
2. Use `git blame` to see who last modified that code
3. Check the commit message and PR context for the change
4. Contact the author or reviewer if needed

### 🚨 Common False Positive: Missing Image Issues

**⚠️ "Missing Image" failures often look scary but are usually harmless false positives.**

**Typical Pattern**:
- Error: `image not found`, `pull failed`, `registry timeout`
- Often affects test setup rather than actual functionality
- May be transient registry connectivity issues

**Quick Assessment**:
```bash
# Check if missing image errors are affecting test setup vs core functionality
grep -A3 -B3 "image.*not found\|pull.*failed" events.txt

# Look for pattern: are these test infrastructure images or core StackRox images?
grep "quay.io\|registry" events.txt | grep -E "test|qa-e2e"
```

**When Missing Image is a False Positive**:
- Affects test setup images (prefetch, qa-e2e containers)
- Transient registry connectivity during cluster creation
- Non-critical sidecar or init containers

**When Missing Image is Real**:
- Affects core StackRox service images (central, scanner, sensor)
- Consistent across multiple test runs
- Related to authentication/authorization with registries

**Resolution**:
- False positives: Usually safe to retry
- Real issues: Escalate to @stackrox/scanner team for registry problems

### 🚨 Red Flags to Look For

- **Rapid pod restarts** in events.txt
- **Database connection failures** in central/scanner logs
- **Certificate expiration/validation errors**
- **Image pull failures** in events
- **Memory/resource constraint warnings**
- **Scanner definition loading failures**
- **Admission webhook timeout errors**

This comprehensive structure enables efficient root cause analysis by providing both high-level status and detailed debugging information for each test failure, along with direct access to the underlying test source code and complete traceability of how each artifact file is created and what information it contains.