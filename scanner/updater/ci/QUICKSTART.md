# CI-Minimal Bundle - Quick Start

## TL;DR

**18 KB bundle embedded in scanner image, 99.991% smaller than full bundle**

```bash
# Use in tests:
export SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip
```

## One-Line Examples

### Scanner E2E Tests
```bash
SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip make -C scanner e2e-test
```

### Local Testing
```bash
docker run -e SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip scanner:latest
```

### GitHub Actions
```yaml
env:
  SCANNER_V4_MATCHER_VULNERABILITIES_URL: file:///etc/scanner/ci-minimal-bundle.zip
```

## What You Get

- ✅ **18 KB** bundle (vs 200 MB)
- ✅ **109 CVEs** from test coverage
- ✅ **No network download** (embedded in image)
- ✅ **Faster startup**

## When to Use

| Scenario | Bundle |
|----------|--------|
| PR/CI tests | ci-minimal (embedded) |
| Local dev | ci-minimal (embedded) |
| Nightly tests | full (GCS) |
| Scale tests | full (GCS) |

## Troubleshooting

**Tests fail with "CVE not found"?**
```bash
# Check if CVE is in allowlist
grep CVE-YYYY-NNNNN scanner/updater/ci/test-cves-allowlist.txt

# Regenerate bundle if needed
./scanner/updater/ci/generate-ci-bundle.sh
```

**Bundle not loading?**
```bash
# Verify bundle exists in image
docker run --rm scanner:latest ls -lh /etc/scanner/ci-minimal-bundle.zip
```

## More Info

- 📖 Full docs: [README.md](./README.md)
- 🔧 Usage guide: [USAGE.md](./USAGE.md)
- 🚀 Integration: [INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md)
