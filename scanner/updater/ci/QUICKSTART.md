# CI-Minimal Bundle - Quick Start

## TL;DR

**18 KB bundle embedded in scanner image, 99.991% smaller than full bundle**

## E2E Tests (Automatic)

Scanner e2e tests use ci-minimal bundle by default:
```bash
helm install scanner ./helmchart
# Automatically uses: vulnerabilitiesUrl: file:///etc/scanner/ci-minimal-bundle.zip
```

For nightly/scale tests with full bundle:
```bash
helm install scanner ./helmchart -f values-full.yaml
```

## Manual Usage

Set environment variable:
```bash
export SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip
```

Or in config YAML:
```yaml
matcher:
  vulnerabilities_url: file:///etc/scanner/ci-minimal-bundle.zip
```

## What You Get

- ✅ **18 KB** bundle (vs 200 MB)
- ✅ **109 CVEs** from test coverage
- ✅ **No network download** (embedded in image)
- ✅ **Faster startup**

## When to Use

| Scenario | Bundle |
|----------|--------|
| PR/CI tests | ci-minimal (default) |
| Local dev | ci-minimal (default) |
| Nightly tests | full (use values-full.yaml) |
| Scale tests | full (use values-full.yaml) |

## More Info

See [README.md](./README.md) for full documentation.
