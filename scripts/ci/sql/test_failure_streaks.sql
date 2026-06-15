-- Find tests with active trailing failure streaks.
--
-- A "trailing streak" means the test is currently failing with no pass
-- in between. The query works inside-out in 3 layers:
--
-- 1. Innermost: number each test run from newest (rn=1) to oldest,
--    per unique (Name, Classname, JobName) group.
-- 2. Middle: for each group, find the position of the first pass
--    (break_at). If a test never passed, break_at is NULL.
-- 3. Outer: keep only failed rows more recent than the first pass
--    (rn < break_at). NULL break_at → COALESCE to 999999 so tests
--    that never passed are included.
--
-- Parameters (BigQuery @-syntax):
--   @days        — lookback window in days
--   @min_streak  — minimum consecutive failures to report
--   @limit       — max rows to return

SELECT
  IF(LENGTH(Name) > 50, CONCAT(RPAD(Name, 47), "..."), Name) AS test_name,
  IF(LENGTH(REPLACE(Classname, "github.com/stackrox/rox/", "")) > 30,
     CONCAT(RPAD(REPLACE(Classname, "github.com/stackrox/rox/", ""), 27), "..."),
     REPLACE(Classname, "github.com/stackrox/rox/", "")) AS suite,
  REGEXP_REPLACE(JobName,
    r"^(periodic-ci-stackrox-stackrox-master-|branch-ci-stackrox-stackrox-(nightlies|master)-)",
    "") AS job,
  COUNT(*) as consecutive_count,
  DATE_DIFF(DATE(MAX(Timestamp)), DATE(MIN(Timestamp)), DAY) + 1 as duration_days
FROM (
  SELECT *,
    MIN(IF(Status = "passed", rn, NULL)) OVER (PARTITION BY Name, Classname, JobName) as break_at
  FROM (
    SELECT Name, Classname, JobName, Status, Timestamp,
      ROW_NUMBER() OVER (PARTITION BY Name, Classname, JobName ORDER BY Timestamp DESC) as rn
    FROM `acs-san-stackroxci.ci_metrics.stackrox_tests`
    WHERE Timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL @days DAY)
      AND Status IN ("passed", "failed")
      AND NOT STARTS_WITH(JobName, "rehearse-")
      AND NOT STARTS_WITH(JobName, "pull-")
      AND NOT CONTAINS_SUBSTR(JobName, "-release-")
      AND NOT CONTAINS_SUBSTR(JobName, "-interop-")
      AND Classname NOT LIKE "CVE-%"
  )
)
WHERE Status = "failed"
  AND rn < COALESCE(break_at, 999999)
GROUP BY Name, Classname, JobName
HAVING COUNT(*) >= @min_streak
  AND DATE(MAX(Timestamp)) = CURRENT_DATE()
ORDER BY consecutive_count DESC, duration_days DESC
LIMIT @limit
