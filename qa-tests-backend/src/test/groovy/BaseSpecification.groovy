import spock.lang.Specification
import OrchestratorManager.OrchestratorMain
import OrchestratorManager.OrchestratorType
import OrchestratorManager.OrchestratorTypes
import spock.lang.Shared
import groovy.util.logging.Slf4j
import org.junit.Rule
import org.junit.rules.TestName
import org.junit.rules.Timeout
import com.jayway.restassured.RestAssured
import testrailIntegration.TestRailconfig


@Slf4j
class BaseSpecification extends Specification {
    @Rule
    def Timeout globalTimeout = new Timeout(200000)
    @Rule
    def TestName name = new TestName()
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
        orchestrator.setup()
    /*    if (isTestrail == true) {
            tc.createTestRailInstance()
            tc.setProjectSectionId("Prevent", "Policies")
            tc.createRun()

        }*/
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