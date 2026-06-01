# Variant Comparison: Finding-as-Advisory vs. Finding-as-CVE

## Overview

Both variants share the same tables (`image_scan_v2`, `scan_components`, `scan_findings`), the same UI, the same REST API endpoints, and the same ingestion hook. They differ in **how advisories are stored** and **where dedup happens**.

| Aspect | Variant 1: Finding = Advisory | Variant 2: Finding = CVE |
|--------|-------------------------------|--------------------------|
| Branch | `dashrews/prototype/finding-as-advisory` | `dashrews/prototype/finding-as-cve` |
| PR | #20852 | TBD |
| scan_findings grain | One row per advisory per component per image | One row per CVE per component per image |
| Advisory storage | First-class columns (`advisoryid`, `sourcename`) | JSONB column (`advisories`) |
| Scanner dedup | Removed — all advisories flow through | Fixed — groups by CVE name, merges advisories |
| PK format | `advisory_id#component_id` | `cve_name#component_id` |
| Est. rows | ~3,265,000 (same as current) | ~3,148,000 (~3.6% fewer) |

## Scanner Mapper Differences

### Variant 1 (finding-as-advisory)
- `dedupeVulns()` call **removed** entirely
- Every advisory flows through as its own `VulnerabilityReport_Vulnerability` entry
- Each entry has `CveName`, `AdvisoryId`, `SourceName` set
- Scanner owns: CVE name resolution only

### Variant 2 (finding-as-cve)
- `dedupeVulns()` call **restored and fixed**
- Groups by resolved CVE name (via `vulnerabilityName()`) instead of raw ClairCore name
- Picks winner by highest `NormalizedSeverity`, then highest CVSS
- Merges all advisories into `AdvisoryDetails` repeated field on the winner
- Scanner owns: CVE name resolution AND advisory merging

## scan_findings Schema Differences

### Variant 1
```
advisory_id   text    -- GHSA-xxx, GO-xxx (indexed, first-class column)
source_name   text    -- "Go Vulnerability DB" (first-class column)
```
No JSONB. Advisory data is queryable via standard SQL.

### Variant 2
```
reserved 2, 20;       -- advisory_id and source_name removed
advisories    jsonb   -- [{id, severity, cvss, source, link}]
```
Advisory data stored as JSONB array. Queryable via JSONB operators but less efficient for filtering.

## Query Differences

### CVE List Page
Both variants need `GROUP BY cvename` to collapse across components and images. Variant 1 also collapses across advisories; Variant 2 doesn't need that because there's only one row per CVE per component.

**Variant 1:**
```sql
SELECT cvename, MAX(severity), MAX(cvss),
       COUNT(DISTINCT imageid), BOOL_OR(isfixable)
FROM scan_findings WHERE state = 0
GROUP BY cvename
-- Groups across: advisories + components + images
```

**Variant 2:**
```sql
SELECT cvename, MAX(severity), MAX(cvss),
       COUNT(DISTINCT imageid), BOOL_OR(isfixable)
FROM scan_findings WHERE state = 0
GROUP BY cvename
-- Groups across: components + images only (no advisory duplication)
```

### Advisory List
**Variant 1:** Simple `SELECT DISTINCT advisoryid, ...` — advisory data is in columns.

**Variant 2:** Requires JSONB extraction: `jsonb_array_elements(advisories) ->> 'id'`. More complex query.

### CVE Detail
**Variant 1:** Multiple rows per CVE returned. API deduplicates advisories by ID.

**Variant 2:** One row per CVE per component. API reads `advisories` JSONB to get advisory list. No dedup needed at API level.

## Tradeoffs

### Variant 1 Advantages
- Simpler pipeline — no dedup logic anywhere, fewer moving parts
- Advisory data is first-class SQL columns — easy to query, index, filter
- No information loss — what scanner produces is exactly what's stored
- Advisory list page is a simple query

### Variant 2 Advantages
- Fewer rows (~3.6% fewer) — less GROUP BY work on CVE list page
- Scanner does the hard work — central just stores what it receives
- One row per CVE per component is conceptually cleaner for most access patterns
- JSONB advisory data is self-contained — no cross-row dedup needed

### Variant 1 Disadvantages
- More rows — GROUP BY has to collapse advisory duplication
- API handler must dedup advisories when building CVE detail response
- Advisory counts in deployment/image queries are inflated by advisory count

### Variant 2 Disadvantages
- JSONB is harder to query/index than first-class columns
- Scanner merge logic adds complexity (winner selection, metadata merging)
- Advisory list page requires JSONB extraction
- If merge logic has bugs, data is lost (merged incorrectly) vs. Variant 1 where raw data is preserved

## What to Benchmark

1. CVE list page query latency (GROUP BY performance with vs. without advisory duplication)
2. CVE detail query latency
3. Advisory list query latency (SQL columns vs. JSONB extraction)
4. Total storage size (rows * row size including JSONB overhead)
5. Ingestion throughput (scan processing time with vs. without merge logic)
