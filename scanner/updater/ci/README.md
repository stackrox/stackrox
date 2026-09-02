# Scanner CI-Minimal Vulnerability Bundle

## Purpose

The ci-minimal bundle contains only vulnerability data needed for CI tests, reducing:
- **Bundle size**: <5MB vs ~200MB (>95% reduction)
- **Download time**: Eliminated (embedded in scanner image)
- **Test startup time**: Faster bundle import

## Contents

**CVEs included**: Only the 109 CVEs explicitly tested in CI test files
- **scanner/e2etests/testdata/**: 94 CVEs (primary source)
- **qa-tests-backend/src/test/groovy/**: 17 CVEs (some overlap with scanner e2e)
- **Combined unique**: 109 CVEs after deduplication
- See `test-cves-allowlist.txt` for complete list

**Sources filtered**:
- `alpine` - Alpine Linux CVEs (test images: alpine:3.13-3.18, nginx:alpine)
- `debian` - Ubuntu/Debian CVEs (test images: ubuntu:16.04-23.10, debian:12.0)
- `osv` - Application CVEs (Jackson, Log4j, Spring, Node.js packages)
- `rhel-vex` - RHEL/UBI CVEs (test images: ubi9, rhacs-collector, jenkins-agent)
- `manual` - Urgent vulnerability fixes

Each source bundle is filtered to include ONLY vulnerability records matching the CVE allowlist.

**Sources excluded** (not tested in CI):
- `suse`, `aws`, `oracle`, `photon` - Minimal test coverage
- `nvd`, `epss`, `cisa-kev` - Enrichment only, not required for detection
- `ubuntu` - Uses debian source data

## Usage

### CI/PR/qa-e2e-tests (fast, uses embedded ci-minimal bundle)

Configure scanner with:
```yaml
matcher:
  vulnerabilities_url: file:///etc/scanner/ci-minimal-bundle.zip
```

Or via environment variable:
```bash
SCANNER_V4_MATCHER_VULNERABILITIES_URL=file:///etc/scanner/ci-minimal-bundle.zip
```

### Nightly/scale tests (comprehensive, uses full GCS bundle)

Configure scanner with:
```yaml
matcher:
  vulnerabilities_url: https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip
```

## Maintenance

### When to regenerate

Regenerate the bundle when:
- New test images added with different OS distributions
- New CVEs added to test assertions
- Scanner updater code changes affect bundle format
- Monthly (to capture updated vulnerability metadata)

### How to regenerate

```bash
# 1. Run analysis to verify sources (optional)
./scanner/updater/ci/analyze-test-sources.sh

# 2. Generate new bundle
./scanner/updater/ci/generate-ci-bundle.sh

# 3. Validate coverage
./scanner/updater/ci/validate-bundle-coverage.sh

# 4. Commit
git add scanner/updater/ci/bundles/ci-minimal/vulnerabilities.zip
git add scanner/updater/ci/test-cves-allowlist.txt
git commit -m "scanner: regenerate ci-minimal vulnerability bundle"
```

### Validation

**Automatic validation** runs on:
- PRs modifying bundle or test data
- Pre-commit hook (optional)

**Manual validation**:
```bash
./scanner/updater/ci/validate-bundle-coverage.sh
```

## Scripts

- `analyze-test-sources.sh` - Analyze test image distribution and CVE coverage
- `extract-test-cves.sh` - Extract CVE allowlist from test files
- `generate-full-bundle.sh` - Generate full bundles from required sources
- `filter-bundle-by-cves.sh` - Filter bundles to only allowlisted CVEs
- `generate-ci-bundle.sh` - Orchestrator script (runs all steps)
- `validate-bundle-coverage.sh` - Validate bundle coverage against tests

## Bundle Generation Process

1. **Extract CVE allowlist** (`extract-test-cves.sh`):
   - Grep for `CVE-YYYY-NNNNN` in scanner/e2etests and qa-tests-backend
   - Output: 109 unique CVEs in `test-cves-allowlist.txt`

2. **Generate source bundles** (`generate-full-bundle.sh`):
   - Export alpine, debian, osv, rhel-vex, manual sources
   - Uses `STACKROX_SCANNER_V4_UPDATER_SOURCES` env var
   - Output: Unfiltered .json.zst files in `bundles/full/`

3. **Filter by CVE** (`filter-bundle-by-cves.sh`):
   - Decompress each source bundle
   - Keep only records where `.Vuln.name` matches allowlist
   - Recompress and package into `bundles/ci-minimal/vulnerabilities.zip`

4. **Validate** (`validate-bundle-coverage.sh`):
   - Extract CVEs from bundle
   - Compare against test CVEs
   - Fail if any test CVE is missing

## Size Comparison

- **Full bundle** (all sources, all CVEs): ~200MB compressed
- **Source-filtered** (5 sources, all CVEs): ~50MB compressed
- **CI-minimal** (5 sources, 109 CVEs): <5MB compressed (**>95% reduction**)

## Troubleshooting

**Bundle generation fails**:
- Ensure `scanner/bin/updater` is built: `make -C scanner bin/updater`
- Check internet connectivity (downloads NVD data)
- Verify manual vulns URL is accessible

**Validation fails (missing CVEs)**:
- Check if new test images added since last generation
- Verify source selection covers all test OS distributions
- Regenerate bundle: `./scanner/updater/ci/generate-ci-bundle.sh`

**Bundle too large (>10MB)**:
- Verify filtering is working: check `filter-bundle-by-cves.sh` output
- Check if test CVE count increased significantly
- Review `test-cves-allowlist.txt` for unexpected entries
