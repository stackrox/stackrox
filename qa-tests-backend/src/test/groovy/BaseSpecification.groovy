import spock.lang.Specification
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import orchestratormanager.OrchestratorTypes
import spock.lang.Shared
import groovy.util.logging.Slf4j
import org.junit.Rule
import org.junit.rules.TestName
import org.junit.rules.Timeout
import com.jayway.restassured.RestAssured
import testrailintegration.TestRailconfig

@Slf4j
class BaseSpecification extends Specification {
    @Rule
    Timeout globalTimeout = new Timeout(200000)
    @Rule
    TestName name = new TestName()
    @Shared
    boolean isTestrail = System.getenv("testrail")
    @Shared
    TestRailconfig tc = new TestRailconfig()
    @Shared
    def resultMap = [:]
    @Shared
    OrchestratorMain orchestrator = OrchestratorType.create(OrchestratorTypes.valueOf(System.getenv("cluster")), "qa")

    def setupSpec() {
        RestAssured.useRelaxedHTTPSValidation()
    /*    if (isTestrail == true) {
            tc.createTestRailInstance()
            tc.setProjectSectionId("Prevent", "Policies")
            tc.createRun()

        }*/
    }
    def setup() {
        orchestrator.setup()
    }
    def cleanupSpec() {
       /* if (isTestrail == true) {
            List<Integer> caseids = new ArrayList<Integer>(resultMap.keySet());
            tc.updateRun(caseids)
            resultMap.each { entry ->
                println "testcaseId: $entry.key status: $entry.value"
                Integer status = Integer.parseInt(entry.key.toString());
                Integer testcaseId = Integer.parseInt(entry.value.toString());
                tc.addStatusForCase(testcaseId, status);
            }

        }*/
    }
}
