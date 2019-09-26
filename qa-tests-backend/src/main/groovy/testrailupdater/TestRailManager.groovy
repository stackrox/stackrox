package testrailupdater

import com.codepine.api.testrail.TestRail
import com.codepine.api.testrail.model.Case
import com.codepine.api.testrail.model.Milestone
import com.codepine.api.testrail.model.Result
import com.codepine.api.testrail.model.Run
import com.codepine.api.testrail.model.Section
import com.codepine.api.testrail.model.Status
import testrailupdater.TestRailUpdater.TestResult
import util.Env

class TestRailManager {
    private static TestRail testRail
    private final static String ENDPOINT = "https://stackroxqa.testrail.net"
    private final static String USERNAME = "k+automation@stackrox.com"
    private final static String PASSWORD = Env.mustGetTestRailPassword()
    private final static String IMAGE_TAG = Env.mustGetImageTag()
    private final static String CI_JOB_NAME = Env.mustGetCiJobName()
    private final static String TEST_RUN_NAME = "${IMAGE_TAG} (${CI_JOB_NAME})"
    private final static int TESTRAIL_API_TIMEOUT = 60 * 1000

    private static projects
    private static suites
    private static sections
    private static List<Case> cases
    private static caseFields
    private static resultFields
    private static List<Status> statuses
    private static List<Milestone> milestones
    private static milestoneName = IMAGE_TAG.find("(\\d{1,3}\\.){3}\\d{1,3}")
    private static milestoneId

    private static run
    private static int projectId
    private static int suiteId
    private static int automationSectionId

    static boolean setupTestRailInstance() {
        if (testRail == null) {
            testRail = TestRail.builder(ENDPOINT, USERNAME, PASSWORD).build()
        }

        def setup = false
        def startTime = System.currentTimeMillis()
        while (!setup && System.currentTimeMillis() - startTime < TESTRAIL_API_TIMEOUT) {
            try {
                projects = testRail.projects().list().execute()
                projectId = projects.find { it.name == "StackRox" }.id

                suites = testRail.suites().list(projectId).execute()
                suiteId = suites.find { it.name == "Master" }.id

                sections = testRail.sections().list(projectId, suiteId).execute()
                automationSectionId = sections.find { it.name == "Automated" }.id

                caseFields = testRail.caseFields().list().execute()

                cases = testRail.cases().list(projectId, suiteId, caseFields).execute()

                resultFields = testRail.resultFields().list().execute()

                statuses = testRail.statuses().list().execute()

                milestones = testRail.milestones().list(projectId).execute()

                println "Setting PROJECT_ID=${projectId}, SUITE_ID=${suiteId}"

                setup = true
            } catch (Exception e) {
                println "Failed to setup TestRail: ${e.toString()}"
                println "Retrying in 5 seconds..."
                sleep 5000
            }
        }

        return setup
    }

    static void updateTestResults(Map<String, List<TestResult>> results) {
        def allCaseIds = []
        for (String spec : results.keySet()) {
            createSectionIfNotExists(spec)
            if (results.get(spec).size() == 0) {
                results.put(spec, updateAllTestsInSpecAsFailed(spec))
            }
            for (TestResult r : results.get(spec)) {
                def id = createCaseIfNotExists(r).getId()
                allCaseIds.add(id)
                r.id = id
            }
        }

        if (milestoneName != null) {
            milestoneId = createMilestoneIfNotExists()
        }
        createNewTestRun(false, allCaseIds)

        for (String spec : results.keySet()) {
            for (TestResult r : results.get(spec)) {
                updateTestResult(r)
            }
        }
    }

    static void createSectionIfNotExists(String specName) {
        def section = sections.find { it.name == specName && it.parentId == automationSectionId }
        if (section == null) {
            println "Test Spec Section \"${specName}\" not found in TestRail. Creating..."
            def created = false
            def startTime = System.currentTimeMillis()
            while (!created && System.currentTimeMillis() - startTime < TESTRAIL_API_TIMEOUT) {
                try {
                    section = testRail.sections().add(
                            projectId,
                            new Section()
                                    .setSuiteId(suiteId)
                                    .setName(specName)
                                    .setParentId(automationSectionId))
                            .execute()
                    sections.add(section)
                    created = true
                } catch (Exception e) {
                    println "failed to create section: ${e.toString()}"
                    println "Retrying in 5 seconds..."
                    sleep 5000
                }
            }

            timeoutCheck(created)
        }
    }

    static Case createCaseIfNotExists(TestResult test) {
        def sectionId = sections.find { it.name == test.suite && it.parentId == automationSectionId }?.id
        def testCase = cases.find { it.title == test.testName && it.sectionId == sectionId }
        if (testCase == null) {
            println "Test Case \"${test.testName}\" not found in TestRail. Creating..."
            def created = false
            def startTime = System.currentTimeMillis()
            while (!created && System.currentTimeMillis() - startTime < TESTRAIL_API_TIMEOUT) {
                try {
                    testCase = testRail.cases().add(
                            sectionId,
                            new Case().setTitle(test.testName),
                            caseFields)
                            .execute()
                    cases.add(testCase)
                    created = true
                } catch (Exception e) {
                    println "failed to create case: ${e.toString()}"
                    println "Retrying in 5 seconds..."
                    sleep 5000
                }
            }

            timeoutCheck(created)
        }
        return testCase
    }

    static int createMilestoneIfNotExists() {
        Milestone milestone = milestones.find { it.name == milestoneName }
        if (milestone == null) {
            println "milestone \"${milestoneName}\" not found in TestRail. Creating..."
            def startTime = System.currentTimeMillis()
            while (System.currentTimeMillis() - startTime < TESTRAIL_API_TIMEOUT) {
                try {
                    milestone = testRail.milestones().add(
                            projectId,
                            new Milestone().setName(milestoneName))
                            .execute()
                    milestones.add(milestone)
                    return milestone.id
                } catch (Exception e) {
                    println "failed to create milestone: ${e.toString()}"
                    println "Retrying in 5 seconds..."
                    sleep 5000
                }
            }

            println "API Timeout trying to create milestone. Milestone will not be used"
        }
        return milestone?.id
    }

    static Run createNewTestRun(Boolean includeAllTestCases = true, List<Object> caseIds = []) {
        Run newRun = new Run()
                .setSuiteId(suiteId)
                .setName(TEST_RUN_NAME)
                .setIncludeAll(includeAllTestCases)
                .setCaseIds(caseIds)
        if (milestoneId != null) {
            newRun.setMilestoneId(milestoneId)
        }

        def created = false
        def startTime = System.currentTimeMillis()
        while (!created && System.currentTimeMillis() - startTime < TESTRAIL_API_TIMEOUT) {
            try {
                println "Creating new test run: ${TEST_RUN_NAME}"
                run = testRail.runs().add(projectId, newRun).execute()
                created = true
            } catch (Exception e) {
                println "Failed to create new test run: ${e.toString()}"
                println "Retrying..."
                sleep 5000
            }
        }

        timeoutCheck(created)
    }

    static boolean updateTestResult(TestResult test) {
        Result result = new Result()
                .setStatusId(statuses.find { it.name == test.passed.toString() }.id)
                .setComment(test.reason)
                .setElapsed(test.elapsed)
        def updated = false
        def startTime = System.currentTimeMillis()
        while (!updated && System.currentTimeMillis() - startTime < TESTRAIL_API_TIMEOUT) {
            try {
                testRail.results().addForCase(run.getId(), test.id, result, resultFields).execute()
                updated = true
            } catch (Exception e) {
                println "Failed to update test run: ${e.toString()}"
                println "Retrying..."
                sleep 5000
            }
        }

        timeoutCheck(updated)
    }

    static List<TestResult> updateAllTestsInSpecAsFailed(String section) {
        List<TestResult> results = []
        def sectionId = sections.find { it.name == section && it.parentId == automationSectionId }?.id
        if (sectionId != null) {
            def sectionCases = cases.findAll { it.sectionId == sectionId }
            for (Case c : sectionCases) {
                results.add(
                        new TestResult(
                                suite: section,
                                testName: c.title,
                                passed: TestRailUpdater.TestResultStatus.FAILED,
                                reason: "")
                )
            }
        } else {
            println "Spec ${section} not found in TestRail - " +
                    "cannot mark all tests as failed as result of classMethod failure!"
        }
        return results
    }

    static void timeoutCheck(boolean state) {
        if (!state) {
            println ("Timeout trying to execute operation.")
        }
    }
}
