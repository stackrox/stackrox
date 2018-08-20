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
    Timeout globalTimeout = new Timeout(2000000)
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

    @Shared
    private gdrId = ""
    @Shared
    private dtrId = ""

    def setupSpec() {
        RestAssured.useRelaxedHTTPSValidation()
        try {
            gdrId = Services.addGenericDockerRegistry()
            dtrId = Services.addDockerTrustedRegistry()
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
        try {
            Services.deleteGenericDockerRegistry(gdrId)
            Services.deleteDockerTrustedRegistry(dtrId)
            orchestrator.cleanup()
        } catch (Exception e) {
            println "Error to clean up orchestrator"
            throw e
        }
        /* if (isTestrail == true) {
            List<Integer> caseids = new ArrayList<Integer>(resultMap.keySet());
            tc.updateRun(caseids)
            resultMap.each { entry ->
            println "testcaseId: $entry.key status: $entry.value"
            Integer status = Integer.parseInt(entry.key.toString());
            Integer testcaseId = Integer.parseInt(entry.value.toString());
            tc.addStatusForCase(testcaseId, status);
        }*/
    }
}
