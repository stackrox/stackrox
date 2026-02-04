---
name: run-e2e-groovy-test
description: Run StackRox E2E Groovy tests locally against a running cluster. Handles environment setup, test configuration, and provides debugging guidance when tests fail.
argument-hint: [test-name]
---

You are an expert at running StackRox E2E tests locally. You guide users through the complete process from environment verification to test execution and debugging.

## Test to Run
${1:The user wants to run test(s): $ARGUMENTS}
${1:else:Ask the user which test(s) to run}

## Workflow

When a user asks to run E2E tests, follow this systematic approach:

### 1. Understand Requirements
- Identify which test(s) to run (specific test class, test method, or full suite)

### 2. Verify Proto Files (Usually Not Needed)

Proto files are usually present from previous builds. Only needed after `./gradlew clean` or in fresh clones.

**If you get build errors** mentioning `scanner/api/v1/*.proto`, run: `make proto-generated-srcs`

### 3. Set Up Test Configuration File

**REQUIRED**: Create `qa-tests-backend/qa-test-settings.properties` with these variables:

```properties
# Required - auth and cluster config
ROX_ADMIN_PASSWORD=<your-central-admin-password>
CLUSTER=K8S  # or OPENSHIFT
POD_SECURITY_POLICIES=false  # true for K8s <1.25

# Required - registry credentials (get from quay.io, NOT Bitwarden)
# Go to https://quay.io/user/<your-username>/?tab=settings -> Generate Encrypted Password
REGISTRY_USERNAME=<your-quay-username>
REGISTRY_PASSWORD=<encrypted-password-from-quay>
```

**Get registry credentials**:
- Visit https://quay.io/user/<your-username>/?tab=settings
- Click "Generate Encrypted Password"
- Use those credentials in the properties file above

**Optional variables** (only needed if specific tests fail):
- GCP credentials and other cloud provider settings - copy from [Bitwarden](https://vault.bitwarden.com/#/vault?itemId=da41ea10-15fa-44e2-988e-af260101b26e) if tests fail requiring them

**Note**: File is in `.gitignore` and won't be committed

### 4. Verify Cluster Prerequisites

Check these cluster-related prerequisites in parallel:
- Stackrox repo is accessible (usually current directory or `$HOME/go/src/github.com/stackrox/stackrox`)
- Java 11+ installed: `java -version`
- Kubernetes cluster: `kubectl cluster-info`
- StackRox deployment: `kubectl get pods -n stackrox`
- Central pod is running and healthy
- Port forward on 8000: `kubectl -n stackrox port-forward svc/central 8000:443`
- Cluster name is "remote" (see Critical Check below)

#### Verify Central Authentication

Test that Central is accessible with the credentials in `qa-test-settings.properties`:

```bash
# Read password from settings file
PASSWORD=`grep ROX_ADMIN_PASSWORD qa-tests-backend/qa-test-settings.properties | cut -d'=' -f2`

# Test authentication
curl -u admin:$PASSWORD -k https://localhost:8000/v1/metadata
# Should return JSON with version info
```

If authentication fails or `ROX_ADMIN_PASSWORD` is not defined, find the password:

**User is typically `admin`. Password locations:**
- **OpenShift**: `deploy/openshift/central-deploy/password`
- **Kubernetes**: `deploy/k8s/central-deploy/password`
- **Common fallbacks**: `admin/admin`, `admin/stackrox`, `stackrox/stackrox`

Check whether these are valid using the above curl command. Once found, update `ROX_ADMIN_PASSWORD` in `qa-test-settings.properties`. If nothing works, ask the user.

#### Critical: Cluster Name Check

**Most tests expect cluster named "remote"** (hardcoded in `ClusterService.groovy:18`).

Check all cluster names:
```bash
curl -k -u <USER>:<PASSWORD> https://localhost:8000/v1/clusters 2>/dev/null | jq -r '.clusters[] | .name'
```

**If "remote" is NOT in the list**: Ask user to choose:
- Option 1: Edit `qa-tests-backend/src/main/groovy/services/ClusterService.groovy` line 18 to match an actual cluster name, then rebuild
- Option 2: Redeploy StackRox with standard script (creates "remote" by default)

**DO NOT proceed** without fixing - tests will fail with "null cluster" errors.

### 5. Check Test-Specific Requirements

Review `qa-tests-backend/README.md` and "Test-specific Notes" section below for:
- Required ConfigMaps/Secrets
- Cluster permissions
- Deployed applications or integrations
- StackRox configuration needs

### 6. Run the Test

**IMPORTANT: Use `qa-tests-backend/gradlew` directly from repo root, NOT `tests/e2e/run-e2e-tests.sh`**

The wrapper script is for CI and often fails locally (Docker issues, missing images, TTY problems).

Run from repository root (ensures correct context and matches CI behavior):

```bash
# Standard test run (uses qa-test-settings.properties for configuration)
qa-tests-backend/gradlew -p qa-tests-backend :test --tests DeclarativeConfigTest

# Run specific test method
qa-tests-backend/gradlew -p qa-tests-backend :test --tests DeclarativeConfigTest --tests='*.testMethodName*'

# Save output for debugging
qa-tests-backend/gradlew -p qa-tests-backend :test --tests DeclarativeConfigTest 2>&1 | tee /tmp/test-output.log

# Run multiple tests
qa-tests-backend/gradlew -p qa-tests-backend :test --tests DeclarativeConfigTest --tests RbacAuthTest
```

**Note**: The test framework automatically loads settings from `qa-test-settings.properties` if it exists. Environment variables can override these settings if needed, but the properties file is the recommended approach.

### 7. Handle Test Failures

If tests fail, investigate:

1. Check test output for error messages
2. Review Central logs: `kubectl logs -n stackrox deploy/central --tail=100`
3. Verify mounted resources (if applicable): `kubectl exec -n stackrox deploy/central -- ls -la /run/stackrox.io/`
4. Check test logs: `qa-tests-backend/build/test-results/` and `qa-tests-backend/build/reports/tests/test/index.html`
5. Ensure previous test runs cleaned up properly
6. **If the failure is not covered in the "Common Issues" section below**, read `qa-tests-backend/README.md` for additional troubleshooting guidance:
   - Search for error messages or test names in the README
   - Look for setup requirements specific to the failing test
   - If you find a solution, apply it or explain it to the user
   - If the README doesn't provide clarity, present what you found and ask the user how to proceed

## Common Issues

### Test hangs during setup / Infinite retry loop with Quay authentication errors
**Symptoms**:
- Test starts but never runs actual test methods
- Repeated warnings every ~3 seconds: `WARN | ImageIntegrationService | Integration test failed: core quay`
- Error message: `http: non-successful response (status=401 body="{\"error\": \"Invalid bearer token format\"}")`
- Test setup phase takes minutes without progress
- Stack traces show `BaseSpecification.setupCoreImageIntegration` repeatedly

**Root cause**: Registry credentials in `qa-test-settings.properties` are missing or invalid.

**Solution**:
1. **Kill the hanging test** (Ctrl+C)

2. **Set registry credentials** in `qa-test-settings.properties`:
   - Get credentials from https://quay.io/user/<your-username>/?tab=settings
   - Click "Generate Encrypted Password"
   - Add to `qa-test-settings.properties`:
     ```properties
     REGISTRY_USERNAME=<your-quay-username>
     REGISTRY_PASSWORD=<encrypted-password-from-quay>
     ```

3. **Rerun the test**:
   ```bash
   qa-tests-backend/gradlew -p qa-tests-backend :test --tests YourTest
   # Should now proceed past setup phase
   ```

**Note**: Most tests (62/67) require valid registry credentials. Without them, tests hang in an infinite retry loop during setup.

### Authentication failures / Tests cannot connect to Central
Test Central authentication (see Step 4 for details). If fails, update `ROX_ADMIN_PASSWORD` in `qa-test-settings.properties`.

### "There is no default cluster" / ClusterService.getClusterId() returns null
Cluster name mismatch - see "Critical: Cluster Name Check" above. Fix by editing `ClusterService.groovy:18` or redeploying.

### Configuration Mismatches

**PodSecurityPolicy errors** (removed in K8s 1.25): Set `POD_SECURITY_POLICIES=false` in `qa-test-settings.properties`

**Wrong cluster type** (OpenShift assertion failures): Set `CLUSTER=OPENSHIFT` or `CLUSTER=K8S` to match your environment

### Connection refused on port 8000
Check port-forward: `ps aux | grep port-forward`, verify Central: `kubectl get pods -n stackrox`, kill stale: `pkill -f 'port-forward.*central'`

### ConfigMap/Secret not found
Check test-specific requirements in "Test-specific Notes" below. Verify: `kubectl get configmaps,secrets -n stackrox`

### Gradle build failures
Verify Java 11+: `java -version`. Clear cache: `rm -rf ~/.gradle/caches && qa-tests-backend/gradlew clean build --refresh-dependencies`

### Resource already exists
Previous test didn't clean up. Manually delete test resources or restart Central.

### Other Issues Not Listed Above
If you encounter an issue not covered here:
1. **Read `qa-tests-backend/README.md`** for additional context and troubleshooting
2. Search the README for keywords from the error message or test name
3. Check for prerequisites or setup steps specific to the failing test
4. If the README provides a solution, apply it or explain it clearly to the user
5. If uncertain, present the relevant information from the README and ask the user how to proceed

## Key Files

- Test source: `qa-tests-backend/src/test/groovy/`
- ClusterService: `qa-tests-backend/src/main/groovy/services/ClusterService.groovy:18` (DEFAULT_CLUSTER_NAME)
- Test results: `qa-tests-backend/build/test-results/test/` (XML) and `.../reports/tests/test/index.html` (HTML)
- README: `qa-tests-backend/README.md`

## Best Practices

1. **Check proto files exist first** (Step 2) - Quick check prevents build errors
2. **Set up `qa-test-settings.properties` with required variables** - See Step 3 for minimal required config
3. **Start with simple tests** to verify environment:
   - `HelpersTest` - Runs in <5 seconds, good smoke test (no cluster needed)
   - `RetryTest` - Runs in <5 seconds (no cluster needed)
   - Then progress to integration tests once you confirm setup works
4. **Match cluster type** - Set `CLUSTER=OPENSHIFT` or `CLUSTER=K8S` to match your actual environment
5. **Verify cluster name is "remote"** before running integration tests (Step 4)
6. **Check for existing port-forwards** before creating new ones
7. **Save test output** with `tee`: `qa-tests-backend/gradlew ... 2>&1 | tee /tmp/test.log`
8. **Run one test initially** before full suites
9. **Review test code** to understand what it does (helps with debugging)
10. **Be patient** - some tests have 5+ minute timeouts
11. **If tests hang during setup** - Check registry credentials immediately (see Common Issues)

## Test-Specific Notes

### Simple Unit Tests (No Cluster Required)

Good for verifying basic environment setup without needing a running cluster:

#### HelpersTest
- **Purpose**: Unit tests for helper utility functions (annotation comparison, etc.)
- **Requirements**: `qa-test-settings.properties` file exists
- **Runtime**: <5 seconds
- **Run**: `qa-tests-backend/gradlew -p qa-tests-backend :test --tests HelpersTest`
- **Note**: Good first test to verify Gradle setup

#### RetryTest
- **Purpose**: Unit tests for retry logic and error handling
- **Requirements**: `qa-test-settings.properties` file exists
- **Runtime**: <5 seconds
- **Run**: `qa-tests-backend/gradlew -p qa-tests-backend :test --tests RetryTest`

### Integration Tests (Cluster Required)

Most tests (62 out of 67) extend `BaseSpecification` and require:
- All prerequisites from Steps 1-4 completed
- `qa-test-settings.properties` with correct CLUSTER setting
- Running cluster with StackRox deployed
- Port-forward to Central on port 8000
- Cluster name is "remote"

Examples: `DeploymentCheck`, `DeploymentTest`, `PolicyConfigurationTest`, `NetworkFlowTest`, `NodeTest`, etc.

**Registry credentials**: Most integration tests warn about invalid registry credentials but can still pass. Tests that specifically need to scan images will fail without valid credentials.

### DeclarativeConfigTest

Requires Central deployed with declarative configuration mounts. The test creates the ConfigMap automatically - you only need to configure the mounts.

**Using Helm**:
```bash
helm upgrade stackrox-central-services ... \
  --set central.declarativeConfiguration.mounts.configMaps={declarative-configurations} \
  --set central.declarativeConfiguration.mounts.secrets={sensitive-declarative-configurations}
kubectl rollout restart deployment/central -n stackrox
```

**Using Operator**: Add to Central CR:
```yaml
spec:
  central:
    declarativeConfiguration:
      configMaps: [{name: "declarative-configurations"}]
      secrets: [{name: "sensitive-declarative-configurations"}]
```

**Note**: The secret doesn't need to exist beforehand (marked `optional: true`). Empty mount directories are sufficient - Central's health system registers both mount points regardless of content.

**Verify both mounts registered**:
```bash
curl -k -u <USER>:<PASSWORD> https://localhost:8000/v1/declarative-config/health | jq '.healths[] | select(.resourceType == "CONFIG_MAP") | .name'
```

**Common failures**: "Expected resource not found" means Central wasn't deployed with declarative config mounts enabled. Redeploy with proper configuration.

### ImageScanningTest

Requires AWS ECR integration credentials.

**Prerequisites**:
- All standard integration test prerequisites
- Environment variable: `AWS_ECR_REGISTRY_NAME` set to your AWS ECR registry ID
- Valid AWS credentials configured

**Common failures**:
- `No value assigned for required key AWS_ECR_REGISTRY_NAME` - Set the environment variable before running
- Most users skip this test unless specifically testing ECR integration
