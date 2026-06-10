# Dedup Correctness Verification

**Date:** June 1, 2026
**Cluster:** ds-06-01-cap-old-speech.ocp.infra.rox.systems
**Branch:** dashrews/prototype/finding-as-advisory (Variant 1)
**Data:** 75 scanned images, 19,960 components, 3,996 findings

## Test Results

| # | Test | Result | Detail |
|---|------|--------|--------|
| 1 | Dual advisories stored as separate rows | PASS | CVE-2026-33186 has GHSA-p77j-4mvh-x3m3 (Critical/9.1) and GO-2026-4762 (Unknown/0) as distinct finding rows |
| 2 | GROUP BY produces one CVE row | PASS | CVE-2026-33186 aggregates to: severity=4, cvss=9.1, 45 images, fixable=true |
| 3 | Cross-component CVEs not deduplicated | PASS | CVE-2024-58251 on busybox/busybox-binsh/ssl_client = 3 separate rows. CVE-2026-33814 on golang.org/x/net (2 versions) + stdlib = separate rows |
| 4 | Multiple GHSA+GO advisory pairs exist | PASS | 10+ CVEs confirmed with exactly 2 advisory IDs each (GHSA + GO pairs) |
| 5 | API CVE list returns deduplicated | PASS | `/v1/scandata/cves` returns one entry per CVE name |
| 6 | API CVE detail shows both advisories | PASS | `/v1/scandata/cves/CVE-2026-33186` returns 2 advisories with different severities, descriptions, and links |
| 7 | No true duplicates | PASS | Zero rows where (cvename, advisoryid, componentid) appears more than once |

## Summary

The finding-as-advisory model correctly:
- Stores each advisory match as a separate row (GHSA and GO entries for the same CVE are distinct findings)
- Groups by CVE name at query time to present one CVE per row in the list view
- Preserves cross-component findings (same CVE on different packages = separate rows)
- Eliminates the original duplicate CVE bug — no same-advisory-same-component duplicates exist
- Exposes advisory-level data (severity differences, data source, description) through the API
