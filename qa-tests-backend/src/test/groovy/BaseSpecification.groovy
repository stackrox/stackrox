import com.google.protobuf.Timestamp
import com.jayway.restassured.RestAssured
import common.Constants
import groovy.util.logging.Slf4j
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.ApiTokenService

import io.stackrox.proto.storage.RoleOuterClass
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import org.junit.Rule
import org.junit.rules.TestName
import org.junit.rules.Timeout
import services.AuthService
import services.BaseService
import services.RoleService
import services.SACService
import spock.lang.Shared
import spock.lang.Specification
import testrailintegration.TestRailconfig
import util.Env

import java.util.concurrent.TimeUnit

@Slf4j
class BaseSpecification extends Specification {

    @Rule
    Timeout globalTimeout = new Timeout(500000, TimeUnit.MILLISECONDS)
    @Rule
    TestName name = new TestName()
    @Shared
    boolean isTestrail = System.getenv("testrail")
    @Shared
    TestRailconfig tc = new TestRailconfig()
    @Shared
    def resultMap = [:]
    @Shared
    OrchestratorMain orchestrator = OrchestratorType.create(
            Env.mustGetOrchestratorType(),
            Constants.ORCHESTRATOR_NAMESPACE
    )
    @Shared
    private ApiTokenService.GenerateTokenResponse allAccessToken

    @Shared
    private testStartTime

    @Shared
    private dtrId = ""

    @Shared
    private boolean stackroxScannerIntegrationDidPreExist

    @Shared
    private roleName = ""

    @Shared
    private String pluginConfigID

    def disableAuthzPlugin() {
        if (pluginConfigID != null) {
            SACService.deleteAuthPluginConfig(pluginConfigID)
        }
        pluginConfigID = null
        if (allAccessToken != null) {
            services.ApiTokenService.revokeToken(allAccessToken.metadata.id)
            allAccessToken = null
        }
    }

    def setupSpec() {
        def startTime = System.currentTimeMillis()
        testStartTime = Timestamp.newBuilder().setSeconds(startTime / 1000 as Long)
                .setNanos((int) ((startTime % 1000) * 1000000)).build()
        RestAssured.useRelaxedHTTPSValidation()
        try {
            dtrId = Services.addDockerTrustedRegistry()
            stackroxScannerIntegrationDidPreExist = Services.deleteAutoRegisteredStackRoxScannerIntegrationIfExists()
            orchestrator.setup()
            orchestrator.createNamespace("default")
        } catch (Exception e) {
            println "Error setting up orchestrator: ${e.message}"
            throw e
        }
        /*    if (isTestrail == true) {
            tc.createTestRailInstance()
            tc.setProjectSectionId("Prevent", "Policies")
            tc.createRun()
        }*/

        def allResources = RoleService.getResources()
        Map<String,RoleOuterClass.Access> resourceAccess = [:]
        allResources.getResourcesList().each { it -> resourceAccess.put(it, RoleOuterClass.Access.READ_WRITE_ACCESS) }
        BaseService.useBasicAuth()
        try {
            def response = SACService.addAuthPlugin()
            pluginConfigID = response.getId()
            println response.toString()
            "Create a test role"
            def testRole = RoleOuterClass.Role.newBuilder()
                    .setName("Test Automation Role")
                    .putAllResourceToAccess(resourceAccess)
                    .build()
            roleName = testRole.name
            RoleService.createRole(testRole)
            allAccessToken = services.ApiTokenService.generateToken("allAccessToken", testRole.name)
            BaseService.useApiToken(allAccessToken.token)
            println AuthService.getAuthStatus().toString()
        } catch (StatusRuntimeException e) {
            println("Unable to enable the authz plugin, defaulting to basic auth: ${e.message}")
        }
    }

    def setup() {
        //Always make sure to revert back to the allAccessToken before each test
        if (allAccessToken == null) {
            println("Using basic auth")
            BaseService.useBasicAuth()
            return
        }
        println("Using the allAccessToken")
        BaseService.useApiToken(allAccessToken.token)
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        try {
            Services.deleteImageIntegration(dtrId)
            if (stackroxScannerIntegrationDidPreExist) {
                Services.addStackroxScannerIntegration()
            }
            orchestrator.cleanup()
        } catch (Exception e) {
            println "Error to clean up orchestrator: ${e.message}"
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
        disableAuthzPlugin()
        RoleService.deleteRole(roleName)
    }

    def cleanup() { }
}
