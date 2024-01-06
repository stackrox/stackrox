package util

import static java.util.UUID.randomUUID

import com.google.cloud.bigquery.BigQuery
import com.google.cloud.bigquery.BigQueryOptions
import com.google.cloud.bigquery.FieldValueList
import com.google.cloud.bigquery.Job
import com.google.cloud.bigquery.JobId
import com.google.cloud.bigquery.JobInfo
import com.google.cloud.bigquery.QueryJobConfiguration
import com.google.cloud.bigquery.TableResult
import common.Constants
import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

@CompileStatic
@Slf4j
class TestMetrics {

    TableResult stableSuites
    TableResult stableTests

    TestMetrics() {
        stableSuites = null
        stableTests = null
    }

    void loadStableSuiteHistory(String ciJobName) {
        log.info("Loading stable suites for ${ciJobName}")
        stableSuites = runQuery("""
                SELECT
                    Classname, FailCount, RunCount
                FROM
                (
                    SELECT
                        Classname, count(*) as RunCount,
                        SUM(IF(Status = "failed", 1, 0)) as FailCount
                    FROM `${Constants.CI_TEST_METRICS_TABLE}`
                    WHERE
                        -- By job
                        ShortName = "${ciJobName}"
                            AND
                        -- Last 90 days
                        DATE(Timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY)
                    GROUP BY Classname
                )
                WHERE
                -- Ignore small counts - new suites
                RunCount > 50
                    AND
                -- 'never' fails
                FailCount = 0
                """)
    }

    void loadStableTestHistory(String ciJobName) {
        log.info("Loading stable tests for ${ciJobName}")
        stableTests = runQuery("""
                SELECT
                    Classname, Name, FailCount, RunCount
                FROM
                (
                    SELECT
                        Classname, Name, count(*) as RunCount,
                        SUM(IF(Status = "failed", 1, 0)) as FailCount
                    FROM `${Constants.CI_TEST_METRICS_TABLE}`
                    WHERE
                        -- By job
                        ShortName = "${ciJobName}"
                            AND
                        -- Last 90 days
                        DATE(Timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY)
                    GROUP BY Classname, Name
                )
                WHERE
                -- ignore small counts - tests with variable names, new tests, modified tests
                RunCount > 50
                    AND
                -- 'never' fails
                FailCount = 0
                """)
    }

    private TableResult runQuery(String query) {
        BigQuery bigquery = BigQueryOptions.getDefaultInstance().getService()
        QueryJobConfiguration queryConfig =
            QueryJobConfiguration.newBuilder(query)
                // Use standard SQL syntax for queries.
                // See: https://cloud.google.com/bigquery/sql-reference/
                .setUseLegacySql(false)
                .build()

        // Create a job ID so that we can safely retry.
        JobId jobId = JobId.of(randomUUID().toString())
        Job queryJob = bigquery.create(JobInfo.newBuilder(queryConfig).setJobId(jobId).build())

        // Wait for the query to complete.
        queryJob = queryJob.waitFor()

        // Check for errors
        if (queryJob == null) {
            throw new RuntimeException("Job no longer exists")
        } else if (queryJob.getStatus().getError() != null) {
            // You can also look at queryJob.getStatus().getExecutionErrors() for all
            // errors, not just the latest one.
            throw new RuntimeException(queryJob.getStatus().getError().toString())
        }

        // Get the results.
        return queryJob.getQueryResults()
    }

    Boolean isSuiteStable(String suiteName) {
        for (FieldValueList row : stableSuites.iterateAll()) {
            if (suiteName == row.get("Classname").getStringValue()) {
                log.debug("${suiteName} is a stable suite")
                return true
            }
        }
        return false
    }

    Boolean isTestStable(String suiteName, String testName) {
        for (FieldValueList row : stableTests.iterateAll()) {
            if (suiteName == row.get("Classname").getStringValue() &&
                testName == row.get("Name").getStringValue()) {
                log.debug("${suiteName}/${testName} is a stable test")
                return true
            }
        }
        return false
    }
}
