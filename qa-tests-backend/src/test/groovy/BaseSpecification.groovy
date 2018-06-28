import com.jayway.restassured.RestAssured
import groovy.util.logging.Slf4j
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import orchestratormanager.OrchestratorTypes
import org.junit.Rule
import org.junit.rules.TestName
import org.junit.rules.Timeout
import spock.lang.Shared
import spock.lang.Specification
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
    OrchestratorMain orchestrator = OrchestratorType.create(OrchestratorTypes.valueOf(System.getenv("CLUSTER")), "qa")

    def setupSpec() {
        RestAssured.useRelaxedHTTPSValidation()

        try {
            orchestrator.setup()
        } catch (Exception e) {
            println "Error setting up orchestrator"
            throw e
        }
    /*    if (isTestrail == true) {
            tc.createTestRailInstance()
            tc.setProjectSectionId("Prevent", "Policies")
            tc.createRun()

        }*/
    }
    def setup() { }

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
