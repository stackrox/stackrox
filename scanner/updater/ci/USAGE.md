# Using CI-Minimal Vulnerability Bundle

The ci-minimal vulnerability bundle (18KB) is embedded in the scanner Docker image at:
```
/etc/scanner/ci-minimal-bundle.zip
```

## Configuration

### Option 1: Environment Variable (Recommended)

Set the environment variable to point to the embedded bundle:

```bash
SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip
```

### Option 2: Config File

Modify scanner configuration YAML:

```yaml
matcher:
  vulnerabilities_url: file:///etc/scanner/ci-minimal-bundle.zip
```

### Option 3: Helm Values (for e2e tests)

Update scanner e2e test deployment:

```yaml
# scanner/e2etests/helmchart/values-ci.yaml
scanner:
  env:
    - name: SCANNER_V4_MATCHER_VULNERABILITIES_URL
      value: file:///etc/scanner/ci-minimal-bundle.zip
```

## Test Scenarios

### CI/PR Tests (Fast)

Use the embedded ci-minimal bundle:
```bash
# Set environment variable in CI workflow
SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip
```

**Benefits**:
- No network download (bundle embedded in image)
- Faster startup (18KB vs 200MB)
- Contains all 109 CVEs tested in CI

### Nightly/Scale Tests (Comprehensive)

Use the full bundle from GCS:
```bash
# Use default or set explicitly
SCANNER_V4_MATCHER_VULNERABILITIES_URL=https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip
```

**Benefits**:
- Full vulnerability dataset
- Detects data load regressions
- Tests real-world scenarios

## Verification

After configuring, verify the bundle loads:

```bash
# Check scanner logs for bundle import
docker logs <scanner-container> | grep -i "vulnerability bundle"

# Verify CVE detection works
# Run scanner e2e tests - they should pass with ci-minimal bundle
```

## File URL Support

The scanner already supports `file://` URLs via the `vulnerabilities_url` config option.
The URL is parsed and the file is loaded directly from the filesystem.

Source: `scanner/config/config.go` - `VulnerabilitiesURL` field accepts any valid URL including `file://`.

## Example: Scanner E2E Tests

```bash
# Deploy scanner with ci-minimal bundle
cd scanner/e2etests
helm install scanner-test ./helmchart \
  --set scanner.env[0].name=SCANNER_V4_MATCHER_VULNERABILITIES_URL \
  --set scanner.env[0].value=file:///etc/scanner/ci-minimal-bundle.zip

# Run tests
# Tests should pass - all 109 CVEs present in bundle
```

## Troubleshooting

### Bundle not found

```
Error: failed to load vulnerability bundle: file not found
```

**Solution**: Ensure the scanner image was built with the embedded bundle.
Check if file exists:
```bash
docker run --rm scanner:latest ls -lh /etc/scanner/ci-minimal-bundle.zip
```

### Missing CVEs

```
Error: CVE-YYYY-NNNNN not found in vulnerability database
```

**Solution**:
1. Check if CVE is in the allowlist: `scanner/updater/ci/test-cves-allowlist.txt`
2. If new test added, regenerate bundle: `./scanner/updater/ci/generate-ci-bundle.sh`
3. For tests requiring CVEs not in allowlist, use full bundle

### Bundle validation failed

**Solution**: Run validation script to check coverage:
```bash
./scanner/updater/ci/validate-bundle-coverage.sh
```
