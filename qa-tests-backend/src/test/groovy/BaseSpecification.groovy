import com.jayway.restassured.RestAssured
import common.Constants

import io.stackrox.proto.api.v1.ApiTokenService
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.RoleOuterClass
import java.security.SecureRandom
import java.util.concurrent.TimeUnit
import objects.K8sServiceAccount
import objects.Secret
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import orchestratormanager.OrchestratorTypes
import org.javers.core.Javers
import org.javers.core.JaversBuilder
import org.javers.core.diff.Diff
import org.javers.core.diff.ListCompareAlgorithm
import org.junit.Rule
import org.junit.rules.TestName
import org.junit.rules.Timeout
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import services.BaseService
import services.ClusterService
import services.ImageIntegrationService
import services.MetadataService
import services.RoleService

import spock.lang.Retry
import spock.lang.Shared
import spock.lang.Specification
import util.Env
import util.Helpers
import util.OnFailure

@Retry(condition = { Helpers.determineRetry(failure) })
@OnFailure(handler = { Helpers.collectDebugForFailure(delegate as Throwable) })
class BaseSpecification extends Specification {

    static final Logger LOG = LoggerFactory.getLogger("test." + BaseSpecification.getSimpleName())

    static final String TEST_IMAGE = "quay.io/rhacs-eng/qa:nginx-1-7-9"

    static final String RUN_ID

    public static final String UNRESTRICTED_SCOPE_ID = "io.stackrox.authz.accessscope.unrestricted"

    static {
        String idStr
        try {
            idStr = new File("/proc/self").getCanonicalFile().getName()
        } catch (Exception e) {
            LOG.warn("Could not determine pid, using a random ID", e)
            idStr = new SecureRandom().nextInt().toString()
        }
        RUN_ID = idStr
    }

    private static boolean globalSetupDone = false

    protected static String allAccessToken = null

    public static strictIntegrationTesting = false

    Map<String, List<String>> resourceRecord = [:]

    private static globalSetup() {
        if (globalSetupDone) {
            return
        }

        LOG.info "Performing global setup"

        if (!Env.IN_CI || Env.get("CIRCLE_TAG")) {
            // Strictly test integration with external services when running in
            // a dev environment or in CI against tagged builds (e.g. nightly builds).
            LOG.info "Will perform strict integration testing (if any is required)"
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
            assert ClusterService.getClusterId(), "There is no default cluster. Check if all pods are running"
            try {
                def metadata = MetadataService.getMetadataServiceClient().getMetadata()
                LOG.info "Testing against:"
                LOG.info metadata.toString()
                LOG.info "isGKE: ${orchestrator.isGKE()}"
                LOG.info "isEKS: ${ClusterService.isEKS()}"
                LOG.info "isOpenShift3: ${ClusterService.isOpenShift3()}"
                LOG.info "isOpenShift4: ${ClusterService.isOpenShift4()}"
                LOG.info "Full cluster deets: ${ClusterService.getCluster().toString()}"
            }
            catch (Exception ex) {
                LOG.info "Cannot connect to central : ${ex.message}"
                LOG.info "Check the test target deployment, auth credentials, kube service proxy, etc."
                throw(ex)
            }
        }

        if (ClusterService.isOpenShift3() || ClusterService.isOpenShift4()) {
            assert Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT,
                    "Set CLUSTER=OPENSHIFT when testing OpenShift"
        }
        else {
            assert Env.mustGetOrchestratorType() == OrchestratorTypes.K8S,
                    "Set CLUSTER=K8S when testing non OpenShift"
        }

        withRetry(30, 1) {
            def allResources = RoleService.getResources()
            Map<String, RoleOuterClass.Access> resourceAccess = [:]
            allResources.getResourcesList().each { res ->
                resourceAccess.put(res, RoleOuterClass.Access.READ_WRITE_ACCESS) }

            String testRoleName = "Test Automation Role - ${RUN_ID}"

            RoleService.deleteRole(testRoleName)
            RoleService.createRoleWithScopeAndPermissionSet(testRoleName, UNRESTRICTED_SCOPE_ID, resourceAccess)

            tokenResp = services.ApiTokenService.generateToken("allAccessToken-${RUN_ID}", testRoleName)
        }

        allAccessToken = tokenResp.token

        addShutdownHook {
            LOG.info "Performing global shutdown"
            BaseService.useBasicAuth()
            BaseService.setUseClientCert(false)
            withRetry(30, 1) {
                services.ApiTokenService.revokeToken(tokenResp.metadata.id)
                if (testRole) {
                    RoleService.deleteRole(testRole.name)
                }
            }
        }

        globalSetupDone = true
    }

    @Rule
    Timeout globalTimeout = new Timeout(
            isRaceBuild() ? 2500 : 800,
            TimeUnit.SECONDS
    )
    @Rule
    TestName name = new TestName()

    @Shared
    Logger log = LoggerFactory.getLogger("test." + this.getClass().getSimpleName())

    @Shared
    OrchestratorMain orchestrator = OrchestratorType.create(
            Env.mustGetOrchestratorType(),
            Constants.ORCHESTRATOR_NAMESPACE
    )

    @Shared
    long orchestratorCreateTime = System.currentTimeSeconds()

    @Shared
    String coreImageIntegrationId = null

    @Shared
    private long testStartTimeMillis

    def setupSpec() {
        log.info("Starting testsuite")

        testStartTimeMillis = System.currentTimeMillis()

        RestAssured.useRelaxedHTTPSValidation()
        globalSetup()

        try {
            orchestrator.setup()
        } catch (Exception e) {
            log.error("Error setting up orchestrator", e)
            throw e
        }
        BaseService.useBasicAuth()
        BaseService.setUseClientCert(false)

        coreImageIntegrationId = ImageIntegrationService.getImageIntegrationByName(
                Constants.CORE_IMAGE_INTEGRATION_NAME)
        if (!coreImageIntegrationId) {
            log.info "Adding core image integration"
            coreImageIntegrationId = ImageIntegrationService.createImageIntegration(
                    ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                            .setName(Constants.CORE_IMAGE_INTEGRATION_NAME)
                            .setType("docker")
                            .addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY)
                            .setDocker(
                                    ImageIntegrationOuterClass.DockerConfig.newBuilder()
                                            .setEndpoint("https://quay.io")
                                            .setUsername(Env.mustGetInCI("REGISTRY_USERNAME", "fakeUsername"))
                                            .setPassword(Env.mustGetInCI("REGISTRY_PASSWORD", "fakePassword"))
                                            .build()
                            ).build()
            )
        }
        if (!coreImageIntegrationId) {
            log.warn "Could not create the core image integration."
            log.warn "Check that REGISTRY_USERNAME and REGISTRY_PASSWORD are valid for quay.io."
        }

        recordResourcesAtSpecStart()
    }

    def recordResourcesAtSpecStart() {
        resourceRecord = [
                "namespaces": orchestrator.getNamespaces(),
                "deployments": orchestrator.getDeployments("default") +
                        orchestrator.getDeployments(Constants.ORCHESTRATOR_NAMESPACE),
        ]
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
        log.info("Starting testcase")

        //Always make sure to revert back to the allAccessToken before each test
        resetAuth()

        if (ClusterService.isEKS() && System.currentTimeSeconds() > orchestratorCreateTime + 600) {
            // Avoid EKS k8s client time out which occurs at approx. 15 minutes.
            orchestrator = OrchestratorType.create(
                    Env.mustGetOrchestratorType(),
                    Constants.ORCHESTRATOR_NAMESPACE
            )
            orchestratorCreateTime = System.currentTimeSeconds()
        }
    }

    def cleanupSpec() {
        log.info("Ending testsuite")

        BaseService.useBasicAuth()
        BaseService.setUseClientCert(false)

        log.info "Removing integration"
        ImageIntegrationService.deleteImageIntegration(coreImageIntegrationId)

        try {
            orchestrator.cleanup()
        } catch (Exception e) {
            log.error("Failed to clean up orchestrator", e)
            throw e
        }

        // https://issues.redhat.com/browse/ROX-9950 -- fails on OSD-on-AWS
        if (orchestrator.isGKE()) {
            compareResourcesAtSpecEnd()
        }
    }

    def compareResourcesAtSpecEnd() {
        Javers javers = JaversBuilder.javers()
                .withListCompareAlgorithm(ListCompareAlgorithm.AS_SET)
                .build()

        List<String> namespaces = orchestrator.getNamespaces()
        Diff diff = javers.compare(resourceRecord["namespaces"], namespaces)
        if (diff.hasChanges()) {
            log.info "There is a difference in namespaces between the start and end of this test spec:"
            log.info diff.prettyPrint()
            throw new TestSpecRuntimeException("Namespaces have changed. Ensure that any namespace created " +
                    "in a test spec is deleted in that test spec.")
        }

        List<String> deployments = orchestrator.getDeployments("default") +
                orchestrator.getDeployments(Constants.ORCHESTRATOR_NAMESPACE)
        diff = javers.compare(resourceRecord["deployments"], deployments)
        if (diff.hasChanges()) {
            log.info "There is a difference in deployments between the start and end of this test spec"
            log.info diff.prettyPrint()
            throw new TestSpecRuntimeException("Deployments have changed. Ensure that any deployments created " +
                    "in a test spec are destroyed in that test spec.")
        }
    }

    def cleanup() {
        log.info("Ending testcase")

        Helpers.resetRetryAttempts()
    }

    static addStackroxImagePullSecret(ns = Constants.ORCHESTRATOR_NAMESPACE) {
        // Add an image pull secret to the qa namespace and also the default service account so the qa namespace can
        // pull stackrox images from dockerhub

        if (!Env.IN_CI && (Env.get("REGISTRY_USERNAME", null) == null ||
                           Env.get("REGISTRY_PASSWORD", null) == null)) {
            // Arguably this should be fatal but for tests that don't pull from docker.io/stackrox it is not strictly
            // necessary.
            LOG.warn "The REGISTRY_USERNAME and/or REGISTRY_PASSWORD env var is missing. " +
                    "(this is ok if your test does not use images from docker.io/stackrox)"
            return
        }

        OrchestratorMain orchestrator = OrchestratorType.create(
                Env.mustGetOrchestratorType(),
                ns
        )
        orchestrator.createImagePullSecret(
                "quay",
                Env.mustGetInCI("REGISTRY_USERNAME", "fakeUsername"),
                Env.mustGetInCI("REGISTRY_PASSWORD", "fakePassword"),
                ns,
                "https://quay.io"
        )
        orchestrator.createImagePullSecret(
                "public-dockerhub",
                "",
                "",
                ns,
                "https://docker.io"
        )
        def sa = new K8sServiceAccount(
                name: "default",
                namespace: ns,
                imagePullSecrets: ["quay", "public-dockerhub"]
        )
        orchestrator.createServiceAccount(sa)
    }

    static addGCRImagePullSecret(ns = Constants.ORCHESTRATOR_NAMESPACE) {
        if (!Env.IN_CI && Env.get("GOOGLE_CREDENTIALS_GCR_SCANNER", null) == null) {
            // Arguably this should be fatal but for tests that don't pull from us.gcr.io it is not strictly necessary
            LOG.warn "The GOOGLE_CREDENTIALS_GCR_SCANNER env var is missing. "+
                    "(this is ok if your test does not use images on us.gcr.io)"
            return
        }

        OrchestratorMain orchestrator = OrchestratorType.create(
                Env.mustGetOrchestratorType(),
                ns
        )

        orchestrator.createImagePullSecret(new Secret(
                name: "gcr-image-pull-secret",
                server: "https://us.gcr.io",
                username: "_json_key",
                password: Env.mustGetInCI("GOOGLE_CREDENTIALS_GCR_SCANNER", "{}"),
                namespace: ns
        ))

        orchestrator.addServiceAccountImagePullSecret(
                "default",
                "gcr-image-pull-secret",
                ns
        )
    }

    def removeGCRImagePullSecret() {
        orchestrator.removeServiceAccountImagePullSecret(
                "default",
                "gcr-image-pull-secret",
                Constants.ORCHESTRATOR_NAMESPACE)
        orchestrator.deleteSecret("gcr-image-pull-secret", Constants.ORCHESTRATOR_NAMESPACE)
    }

    static Boolean isRaceBuild() {
        return Env.get("IS_RACE_BUILD", null) == "true" || Env.CI_JOBNAME == "race-condition-tests"
    }

}

class TestSpecRuntimeException extends RuntimeException {
    TestSpecRuntimeException(String message) {
        super(message)
    }
}
