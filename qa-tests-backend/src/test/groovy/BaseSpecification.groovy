import com.jayway.restassured.RestAssured
import common.Constants
import groovy.util.logging.Slf4j
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.ApiTokenService
import io.stackrox.proto.storage.RoleOuterClass
import objects.K8sServiceAccount
import objects.Secret
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import orchestratormanager.OrchestratorTypes
import org.junit.Rule
import org.junit.rules.TestName
import org.junit.rules.Timeout
import services.BaseService
import services.ClusterService
import services.MetadataService
import services.RoleService
import services.SACService
import spock.lang.Retry
import spock.lang.Shared
import spock.lang.Specification
import util.Env
import util.Helpers
import util.OnFailure

import java.security.SecureRandom
import java.util.concurrent.TimeUnit
import java.text.SimpleDateFormat

@Slf4j
@Retry(condition = { Helpers.determineRetry(failure) })
@OnFailure(handler = { Helpers.collectDebugForFailure(delegate as Throwable) })
class BaseSpecification extends Specification {

    static final String RUN_ID

    static {
        String idStr
        try {
            idStr = new File("/proc/self").getCanonicalFile().getName()
        } catch (Exception ignored) {
            println "Could not determine pid, using a random ID"
            idStr = new SecureRandom().nextInt().toString()
        }
        RUN_ID = idStr
    }

    private static boolean globalSetupDone = false

    protected static String allAccessToken = null

    public static strictIntegrationTesting = false

    private static globalSetup() {
        if (globalSetupDone) {
            return
        }

        println "Performing global setup"

        if (!Env.IN_CI || Env.get("CIRCLE_TAG")) {
            // Strictly test integration with external services when running in
            // a dev environment or in CI against tagged builds (e.g. nightly builds).
            println "Will perform strict integration testing (if any is required)"
            strictIntegrationTesting = true
        }

        OrchestratorMain orchestrator = OrchestratorType.create(
                Env.mustGetOrchestratorType(),
                Constants.ORCHESTRATOR_NAMESPACE
        )

        orchestrator.createNamespace(Constants.ORCHESTRATOR_NAMESPACE)

        addStackroxImagePullSecret()
        addGCRImagePullSecret()

        RoleOuterClass.Role testRole = null
        ApiTokenService.GenerateTokenResponse tokenResp = null

        BaseService.useBasicAuth()
        BaseService.setUseClientCert(false)

        withRetry(10, 1) {
            try {
                def metadata = MetadataService.getMetadataServiceClient().getMetadata()
                println "Testing against:"
                println metadata
                println "isGKE: ${orchestrator.isGKE()}"
                println "isEKS: ${ClusterService.isEKS()}"
            }
            catch (Exception ex) {
                println "Check the test target deployment, auth credentials, kube service proxy, etc."
                throw(ex)
            }
        }

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
    Timeout globalTimeout = new Timeout(
            Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT ? 1000 : 500,
            TimeUnit.SECONDS
    )
    @Rule
    TestName name = new TestName()
    @Shared
    OrchestratorMain orchestrator = OrchestratorType.create(
            Env.mustGetOrchestratorType(),
            Constants.ORCHESTRATOR_NAMESPACE
    )

    @Shared
    private long testStartTimeMillis

    @Shared
    private String pluginConfigID

    def disableAuthzPlugin() {
        if (pluginConfigID != null) {
            SACService.deleteAuthPluginConfig(pluginConfigID)
        }
        pluginConfigID = null
    }

    def setupSpec() {
        def date = new Date()
        def sdf = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US)
        println "${sdf.format(date)} Starting testsuite"

        testStartTimeMillis = System.currentTimeMillis()

        RestAssured.useRelaxedHTTPSValidation()
        globalSetup()

        try {
            orchestrator.setup()
        } catch (Exception e) {
            e.printStackTrace()
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
        def date = new Date()
        def sdf = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US)
        println "${sdf.format(date)} Starting testcase"

        //Always make sure to revert back to the allAccessToken before each test
        resetAuth()
    }

    def cleanupSpec() {
        def date = new Date()
        def sdf = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US)
        println "${sdf.format(date)} Ending testsuite"

        BaseService.useBasicAuth()
        BaseService.setUseClientCert(false)
        try {
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

    def cleanup() {
        def date = new Date()
        def sdf = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US)
        println "${sdf.format(date)} Ending testcase"

        Helpers.resetRetryAttempts()
    }

    static addStackroxImagePullSecret() {
        // Add an image pull secret to the qa namespace and also the default service account so the qa namespace can
        // pull stackrox images from dockerhub

        if (!Env.IN_CI && (Env.get("REGISTRY_USERNAME", null) == null ||
                           Env.get("REGISTRY_PASSWORD", null) == null)) {
            // Arguably this should be fatal but for tests that don't pull from docker.io/stackrox it is not strictly
            // necessary.
            println "WARNING: The REGISTRY_USERNAME and/or REGISTRY_PASSWORD env var is missing. " +
                    "(this is ok if your test does not use images from docker.io/stackrox)"
            return
        }

        OrchestratorMain orchestrator = OrchestratorType.create(
                Env.mustGetOrchestratorType(),
                Constants.ORCHESTRATOR_NAMESPACE
        )
        orchestrator.createImagePullSecret(
                "stackrox",
                Env.mustGetInCI("REGISTRY_USERNAME", "fakeUsername"),
                Env.mustGetInCI("REGISTRY_PASSWORD", "fakePassword"),
                Constants.ORCHESTRATOR_NAMESPACE
        )
        def sa = new K8sServiceAccount(
                name: "default",
                namespace: Constants.ORCHESTRATOR_NAMESPACE,
                imagePullSecrets: ["stackrox"]
        )
        orchestrator.createServiceAccount(sa)
    }

    static addGCRImagePullSecret() {
        if (!Env.IN_CI && Env.get("GOOGLE_CREDENTIALS_GCR_SCANNER", null) == null) {
            // Arguably this should be fatal but for tests that don't pull from us.gcr.io it is not strictly necessary
            println "WARNING: The GOOGLE_CREDENTIALS_GCR_SCANNER env var is missing. "+
                    "(this is ok if your test does not use images on us.gcr.io)"
            return
        }

        OrchestratorMain orchestrator = OrchestratorType.create(
                Env.mustGetOrchestratorType(),
                Constants.ORCHESTRATOR_NAMESPACE
        )

        orchestrator.createImagePullSecret(new Secret(
                name: "gcr-image-pull-secret",
                server: "https://us.gcr.io",
                username: "_json_key",
                password: Env.mustGetInCI("GOOGLE_CREDENTIALS_GCR_SCANNER", "{}"),
                namespace: Constants.ORCHESTRATOR_NAMESPACE
        ))

        orchestrator.addServiceAccountImagePullSecret(
                "default",
                "gcr-image-pull-secret",
                Constants.ORCHESTRATOR_NAMESPACE
        )
    }

    def removeGCRImagePullSecret() {
        orchestrator.removeServiceAccountImagePullSecret(
                "default",
                "gcr-image-pull-secret",
                Constants.ORCHESTRATOR_NAMESPACE)
        orchestrator.deleteSecret("gcr-image-pull-secret", Constants.ORCHESTRATOR_NAMESPACE)
    }
}
