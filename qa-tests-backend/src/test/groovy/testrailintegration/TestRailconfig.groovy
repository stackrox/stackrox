package testrailintegration

import com.codepine.api.testrail.TestRail
import com.codepine.api.testrail.TestRail.Projects
import com.codepine.api.testrail.model.Project
import com.codepine.api.testrail.model.Result
import com.codepine.api.testrail.model.Run
import com.codepine.api.testrail.model.CaseField
import com.codepine.api.testrail.model.Case
import com.codepine.api.testrail.model.Section

import groovy.util.logging.Slf4j

import java.text.ParseException

@Slf4j
class TestRailconfig {
    //Create the Test Rail Instance:
    private static TestRail testRail
    private final static String ENDPOINT = "https://stackroxqa.testrail.net"
    private final static String USERNAME = System.getenv("username")
    private final static String PASSWORD = System.getenv("password")
    public static final int TEST_CASE_PASSED_STATUS = 1
    public static final int TEST_CASE_FAILED_STATUS = 5
    private static final Map<String, Integer> CASES_TO_ADD = [:]

    static TestRail createTestRailInstance() {
        if (testRail == null) {
            testRail = TestRail.builder(ENDPOINT, USERNAME, PASSWORD).build()
        }
        return testRail
    }

    //Add  get and set methods  for project id and suite id.

    private static int projectId
    private static int suiteId
    private static int sectionId

    static int getProjectId() {
        return projectId
    }

    static void setProjectId(int projectId) {
        this.projectId = projectId
    }

    static int getSuiteId() {
        return suiteId
    }

    static void setSuiteId(int suiteId) {
        this.suiteId = suiteId
    }

    static int getSectionId() {
        return sectionId
    }

    static void setSectionId(int sectionId) {
        this.sectionId = sectionId
    }

    static void setProjectSectionId(String projectName, String sectionName) {
        try {
            Projects projects = testRail.projects()
            List projectList = projects.list().execute()
            int pid = 0
            int sid = 0
            for (Project project : projectList) {
                if (project.getName().equals(projectName)) {
                    pid = project.getId()
                    setProjectId(pid)
                    System.out.println(pid)
                    break
                }
            }
            if (pid != 0) {
                List sectionList = testRail.sections().list(pid).execute()
                for (Section s : sectionList) {
                    String sName = s.getName()
                    if (sName.equals(sectionName)) {
                        sid = s.getId()
                        setSectionId(sid)
                        System.out.println(sid)
                    }
                }
            }
        }
        catch (Exception e) {
            e.printStackTrace()
        }
    }

    //Create new Run:  assign all test case as false.
    public static Run run

    static Run getRun() {
        return run
    }

    static void setRun(Run run) {
        this.run = run
    }

// ***** Create Run Function *********
    static void createRunforAll() throws ParseException {
        String runName = "Automation TestRun"
        try {
            List<CaseField> customCaseFields = testRail.caseFields().list().execute()
            List<Case> cases = testRail.cases().list(projectId, customCaseFields).execute()
            for (Case c : cases) {
                if (c.sectionId == sectionId) {
                    CASES_TO_ADD.put(c.title, c.id)
                }
            }
            List<Integer> caseIds = CASES_TO_ADD.values().asList()
            run = new Run()
            run = testRail.runs().add(getProjectId(), run.setName(runName).setCaseIds(caseIds)).execute()
            setRun(run)
        }

        catch (Exception e) {
            e.printStackTrace()
        }
    }

    static void createRun() throws ParseException {
        String runName = "Automation TestRun"
        try {
            run = new Run()
            run = testRail.runs().add(getProjectId(), run.setSuiteId(getSuiteId()).setName(runName)
                    .setIncludeAll(false)).execute()
            setRun(run)
        }
        catch (Exception e) {
            e.printStackTrace()
        }
    }

    //To add test case ids at run time
    static void updateRun(List<Integer> caseIdString) {
        List<Integer> caseIds = new ArrayList<Integer>()
        try {
            for (int caseId : caseIdString) {
                caseIds.add(caseId)
            }
            getRun().setCaseIds(caseIds)
            testRail.runs().update(getRun()).execute()
        }
        catch (Exception e) {
            e.printStackTrace()
        }
    }

    // Add Result for test case in current run
    static void addResult(String comment, int caseId) {
        try {
            if (null != testRail()) {
                List customResultFields = testRail.resultFields().list().execute()
                testRail.results()
                        .addForCase(getRun().getId(), caseId, new Result().setComment(comment), customResultFields)
                        .execute()
            }
        }
        catch (Exception e) {
            e.printStackTrace()
        }
    }

    // Add final result for a test case i.e. status like pass/fail/skipped

    static void addStatusForCase(int statusId, int caseId) {
        try {
            List customResultFields = testRail.resultFields().list().execute()
            testRail.results()
                    .addForCase(getRun().getId(), caseId, new Result().setStatusId(statusId), customResultFields)
                    .execute()
        }
        catch (Exception e) {
            e.printStackTrace()
        }
    }

    static void closeRun() {
        try {
            testRail().runs().close(getRun().getId()).execute()
        }
        catch (Exception e) {
            e.printStackTrace()
        }
    }

    static int verifyAndAdd(List<String> actual, String expected, int caseIds) {
        caseIds // Temporarily so that it isn't unused (to appease the linter).
        try {
            assert actual.contains(expected)
            return TEST_CASE_PASSED_STATUS
            // addStatusForCase(TEST_CASE_PASSED_STATUS,caseId)
        }

        catch (AssertionError e) {
            //addStatusForCase(TEST_CASE_FAILED_STATUS,caseId)
            return TEST_CASE_FAILED_STATUS
        }
    }
}
