-- CVE Prototype Benchmark Queries
-- Run against a Postgres instance with realistic scan data.
-- Each query uses EXPLAIN ANALYZE to capture planning and execution times.

-- 1. CVE List Page (GROUP BY, unfiltered)
EXPLAIN ANALYZE
SELECT cve_name, MAX(severity) AS severity, MAX(cvss) AS cvss,
       COUNT(DISTINCT image_id) AS image_count, BOOL_OR(is_fixable) AS fixable,
       MIN(first_system_occurrence) AS first_seen
FROM scan_findings
WHERE state = 0
GROUP BY cve_name
ORDER BY MAX(severity) DESC, MAX(cvss) DESC
LIMIT 20;

-- 2. CVE List Page with severity + fixable filter
EXPLAIN ANALYZE
SELECT cve_name, MAX(severity) AS severity, MAX(cvss) AS cvss,
       COUNT(DISTINCT image_id) AS image_count, BOOL_OR(is_fixable) AS fixable,
       MIN(first_system_occurrence) AS first_seen
FROM scan_findings
WHERE state = 0 AND is_fixable = true AND severity >= 3
GROUP BY cve_name
ORDER BY MAX(severity) DESC, MAX(cvss) DESC
LIMIT 20;

-- 3. CVE Detail: all findings for a specific CVE
EXPLAIN ANALYZE
SELECT advisory_id, cve_name, severity, cvss, source_name, data_source, description
FROM scan_findings
WHERE cve_name = 'CVE-2024-1234';

-- 4. Image Detail: findings for a specific image joined with components
EXPLAIN ANALYZE
SELECT f.cve_name, f.advisory_id, f.severity, f.cvss, f.is_fixable, f.fixed_by,
       c.name, c.version, c.source
FROM scan_findings f
JOIN scan_components c ON f.component_id = c.id
WHERE f.image_id = 'sha256:placeholder'
ORDER BY f.severity DESC, f.cvss DESC;

-- 5. Deployment count: deployments affected by any non-deferred finding
EXPLAIN ANALYZE
SELECT COUNT(DISTINCT d.id)
FROM deployments d
JOIN deployments_containers dc ON d.id = dc.deployments_id
JOIN scan_components c ON dc.image_id = c.image_id
JOIN scan_findings f ON c.id = f.component_id
WHERE f.severity >= 1 AND f.state = 0;

-- 6. CVE count: total distinct CVEs in observed state
EXPLAIN ANALYZE
SELECT COUNT(DISTINCT cve_name) FROM scan_findings WHERE state = 0;

-- 7. Table sizes (no EXPLAIN ANALYZE needed)
SELECT
    relname AS table_name,
    pg_size_pretty(pg_total_relation_size(oid)) AS total_size,
    pg_size_pretty(pg_relation_size(oid)) AS table_size,
    pg_size_pretty(pg_indexes_size(oid)) AS index_size,
    reltuples::bigint AS row_estimate
FROM pg_class
WHERE relname IN ('image_scan_v2', 'scan_components', 'scan_findings')
ORDER BY pg_total_relation_size(oid) DESC;
