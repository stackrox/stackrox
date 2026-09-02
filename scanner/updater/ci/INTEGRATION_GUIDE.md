# Integrating CI-Minimal Bundle into Tests

## Quick Start

The ci-minimal vulnerability bundle is now embedded in the scanner Docker image.
To use it in tests, simply set the environment variable:

```bash
export SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip
```

## Integration Examples

### 1. Scanner E2E Tests

**Option A: Environment Variable (Simplest)**

```bash
cd scanner/e2etests
SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip \
  ./run-e2e-tests.sh
```

**Option B: Helm Values**

Create `helmchart/values-ci.yaml`:
```yaml
scanner:
  extraEnv:
    - name: SCANNER_V4_MATCHER_VULNERABILITIES_URL
      value: file:///etc/scanner/ci-minimal-bundle.zip
```

Then deploy:
```bash
helm install scanner-test ./helmchart -f helmchart/values-ci.yaml
```

### 2. QA Backend Tests (Groovy)

Update test setup to pass environment variable to scanner deployment:

```groovy
// In Kubernetes.groovy or similar
def scannerEnv = [
    [name: "SCANNER_V4_MATCHER_VULNERABILITIES_URL",
     value: "file:///etc/scanner/ci-minimal-bundle.zip"]
]
```

### 3. GitHub Actions CI Workflow

```yaml
- name: Run scanner e2e tests with ci-minimal bundle
  env:
    SCANNER_V4_MATCHER_VULNERABILITIES_URL: file:///etc/scanner/ci-minimal-bundle.zip
  run: |
    make -C scanner e2e-test
```

### 4. Local Development

```bash
# Start scanner with embedded bundle
docker run -d \
  --name scanner-test \
  -e SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip \
  -e SCANNER_DB_HOST=postgres \
  scanner:latest

# Verify bundle loaded
docker logs scanner-test 2>&1 | grep -i "vulnerability"
```

## Test Strategy Recommendation

### Use CI-Minimal Bundle For:
- ✅ PR/CI tests (fast feedback)
- ✅ Unit/integration tests
- ✅ Local development
- ✅ Quick validation

### Use Full Bundle For:
- ✅ Nightly tests (detect data load issues)
- ✅ Scale tests (test with real data volume)
- ✅ Performance benchmarks
- ✅ Production deployment validation

## Verification Steps

After integrating, verify:

1. **Bundle loads successfully**:
   ```bash
   docker logs <scanner> | grep "loaded.*vulnerabilities"
   ```

2. **Tests pass**:
   ```bash
   # All 109 test CVEs should be detected
   make -C scanner e2e-test
   ```

3. **Size/speed improvement**:
   ```bash
   # Compare startup time
   time docker run --rm scanner:latest scanner --version
   ```

## Transition Plan

1. **Week 1**: Test ci-minimal bundle in local environment
2. **Week 2**: Deploy to PR CI tests
3. **Week 3**: Monitor test stability and timing
4. **Week 4**: Expand to all CI test suites (keep nightlies on full bundle)

## Rollback

If issues occur, revert to full bundle:

```bash
# Remove or comment out the environment variable
# unset SCANNER_V4_MATCHER_VULNERABILITIES_URL

# Or explicitly set to full bundle
export SCANNER_V4_MATCHER_VULNERABILITIES_URL=https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip
```

## Troubleshooting

See [USAGE.md](./USAGE.md) for detailed troubleshooting steps.

## Maintenance

When adding new test images or CVEs:

1. Update tests to include new CVE assertions
2. Regenerate bundle:
   ```bash
   ./scanner/updater/ci/generate-ci-bundle.sh
   ```
3. Commit updated bundle:
   ```bash
   git add scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip
   git commit -m "scanner: update ci-minimal bundle with new test CVEs"
   ```
4. Rebuild scanner image to embed new bundle
