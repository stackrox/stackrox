# CVE Redesign Prototype

## Documents (read in this order)

| Document | What it covers |
|----------|----------------|
| [cve-vision.md](cve-vision.md) | Original problem statement and goals for the CVE redesign |
| [cve-duplicate-investigation.md](cve-duplicate-investigation.md) | Root cause analysis of the duplicate CVE problem — data flow trace from ClairCore through scanner mapper to central storage |
| [cve-redesign-design.md](cve-redesign-design.md) | Approved design spec — data model, scanner changes, pipeline, UI/UX, exports, rollout phases |
| [cve-prototype-plan.md](cve-prototype-plan.md) | Implementation plan with task breakdown for both prototype branches |
| [benchmarks/](benchmarks/) | SQL benchmark queries and runner script for performance comparison |

## Prototype Summary

This branch (`dashrews/prototype/finding-as-advisory`) implements **Variant 1: Finding = Advisory**.

Each advisory match from scanner is stored as its own finding row. CVE name is a grouping key resolved at query time via `GROUP BY`. There is no dedup logic — every advisory flows through as a legitimate, separate finding.

### Key Changes

**Scanner** (`pkg/scannerv4/mappers/`)
- Resolves CVE names and populates `CveName`, `AdvisoryId`, `SourceName` on each Vulnerability output
- Removes `dedupeVulns()` — all advisories flow through

**Storage** (3 new tables)
- `image_scan_v2` — scan metadata (scanner version, bundle version, data sources)
- `scan_components` — packages found during scan
- `scan_findings` — one row per advisory per component per image

**Central** (`pkg/scanners/scannerv4/convert.go`)
- Simplified to 1:1 passthrough using explicit `cve_name` from scanner

**REST API** (`central/scandata/api/`)
- `GET /v1/scandata/cves` — CVE list with GROUP BY aggregation
- `GET /v1/scandata/cves/{cveName}` — CVE detail with advisories, components, images
- `GET /v1/scandata/advisories` — advisory list
- `GET /v1/scandata/deployments` — deployment list with CVE counts
- `GET /v1/scandata/deployments/{id}` — deployment detail with per-image CVE summary
- `GET /v1/scandata/images/{imageId}/findings` — findings for an image

**UI** (`ui/apps/platform/src/Containers/Vulnerabilities/proto/`)
- Accessible via sidebar: Vulnerability Management → CVE Prototype
- Three tabs: CVEs, Advisories, Deployments
- CVE detail page with three layout variants (flow, tabs, collapsible) selectable via toggle
- Per-image expandable rows showing which advisory IDs matched each component

### What's NOT in this prototype

- Migration of existing data (new tables populated by new scans only)
- Dual-write / feature flag infrastructure
- Policy engine changes
- Materialized summary table (Approach C — only if GROUP BY is slow)
- Exception/deferral system changes
- Branch 2 (finding-as-cve variant with JSONB advisories)

### Definition of CVE Uniqueness

A CVE instance is unique by: **(cve_name, component_name, component_version, component_source, image_id)**.

Multiple advisories (GHSA, GO vulndb, RHSA) for the same CVE on the same component in the same image are separate findings — not duplicates.
