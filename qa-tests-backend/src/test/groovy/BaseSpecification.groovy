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
import services.BaseService
import services.ImageIntegrationService
import services.RoleService
import services.SACService
import spock.lang.Shared
import spock.lang.Specification
import testrailintegration.TestRailconfig
import util.Env

import java.security.SecureRandom
import java.util.concurrent.TimeUnit

@Slf4j
class BaseSpecification extends Specification {

    static final String RUN_ID

    static {
        String idStr = null
        try {
            idStr = new File("/proc/self").getCanonicalFile().getName()
        } catch (Exception ex) {
            println "Could not determine pid, using a random ID"
            idStr = new SecureRandom().nextInt().toString()
        }
        RUN_ID = idStr
    }

    private static boolean globalSetupDone = false

    private static String allAccessToken = null

    private static globalSetup() {
        if (globalSetupDone) {
            return
        }

        println "Performing global setup"

        RoleOuterClass.Role testRole = null
        ApiTokenService.GenerateTokenResponse tokenResp = null

        BaseService.useBasicAuth()
        BaseService.setUseClientCert(false)

        withRetry(30, 1) {
            def allResources = RoleService.getResources()
            Map<String, RoleOuterClass.Access> resourceAccess = [:]
            allResources.getResourcesList().each { res ->
                resourceAccess.put(res, RoleOuterClass.Access.READ_WRITE_ACCESS) }

            testRole = RoleOuterClass.Role.newBuilder()
                    .setName("Test Automation Role - ${RUN_ID}")
                    .putAllResourceToAccess(resourceAccess)
                    .build()

            RoleService.deleteRole(testRole.name)
            RoleService.createRole(testRole)

            tokenResp = services.ApiTokenService.generateToken("allAccessToken-${RUN_ID}", testRole.name)
        }

        allAccessToken = tokenResp.token

        addShutdownHook {
            println "Performing global shutdown"
            BaseService.useBasicAuth()
            BaseService.setUseClientCert(false)
            withRetry(30, 1) {
                services.ApiTokenService.revokeToken(tokenResp.metadata.id)
                RoleService.deleteRole(testRole.name)
            }
        }

        globalSetupDone = true
    }

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
    private long testStartTimeMillis

    @Shared
    private dtrId = ""

    @Shared
    private boolean stackroxScannerIntegrationDidPreExist

    @Shared
    private String pluginConfigID

    def disableAuthzPlugin() {
        if (pluginConfigID != null) {
            SACService.deleteAuthPluginConfig(pluginConfigID)
        }
        pluginConfigID = null
    }

    def setupSpec() {
        testStartTimeMillis = System.currentTimeMillis()

        RestAssured.useRelaxedHTTPSValidation()
        globalSetup()

        try {
            dtrId = ImageIntegrationService.addDockerTrustedRegistry()
            stackroxScannerIntegrationDidPreExist =
                    ImageIntegrationService.deleteAutoRegisteredStackRoxScannerIntegrationIfExists()
            orchestrator.setup()
        } catch (Exception e) {
            println "Error setting up orchestrator: ${e.message}"
            throw e
        }
        /*    if (isTestrail == true) {
            tc.createTestRailInstance()
            tc.setProjectSectionId("Prevent", "Policies")
            tc.createRun()
        }*/
        BaseService.useBasicAuth()
        BaseService.setUseClientCert(false)
        try {
            def response = SACService.addAuthPlugin()
            pluginConfigID = response.getId()
            println response.toString()
        } catch (StatusRuntimeException e) {
            println("Unable to enable the authz plugin, defaulting to basic auth: ${e.message}")
        }
    }

    def resetAuth() {
        BaseService.setUseClientCert(false)
        if (allAccessToken) {
            BaseService.useApiToken(allAccessToken)
        } else {
            BaseService.useBasicAuth()
        }
    }

    def setup() {
        //Always make sure to revert back to the allAccessToken before each test
        resetAuth()
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        BaseService.setUseClientCert(false)
        try {
            ImageIntegrationService.deleteImageIntegration(dtrId)
            if (stackroxScannerIntegrationDidPreExist) {
                ImageIntegrationService.addStackroxScannerIntegration()
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
    }

    def cleanup() { }
}
