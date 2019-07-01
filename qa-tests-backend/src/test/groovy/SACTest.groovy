import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import objects.Deployment
import org.junit.experimental.categories.Category
import services.DeploymentService
import services.ApiTokenService
import services.SACService
import services.BaseService
import spock.lang.Shared
import spock.lang.Unroll
import groups.BAT

class SACTest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX_NAMESPACE_QA1 = "sac-deploymentnginx-qa1"
    static final private String DEPLOYMENTNGINX_NAMESPACE_QA2 = "sac-deploymentnginx-qa2"
    static final private Deployment DEPLOYMENT_QA1 = new Deployment()
            .setName(DEPLOYMENTNGINX_NAMESPACE_QA1)
            .setImage("nginx:1.7.9")
            .addPort(22, "TCP")
            .addAnnotation("test", "annotation")
            .setEnv(["CLUSTER_NAME": "main"])
            .setNamespace("test-qa1")
            .addLabel("app", "test")
    static final private Deployment DEPLOYMENT_QA2 = new Deployment()
            .setName(DEPLOYMENTNGINX_NAMESPACE_QA2)
            .setImage("nginx:1.7.9")
            .addPort(22, "TCP")
            .addAnnotation("test", "annotation")
            .setEnv(["CLUSTER_NAME": "main"])
            .setNamespace("test-qa2")
            .addLabel("app", "test")

    static final private List<Deployment> DEPLOYMENTS = [DEPLOYMENT_QA1, DEPLOYMENT_QA2,]
    @Shared
    private String pluginConfigID

    def setupSpec() {
        BaseService.useBasicAuth()
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
        def response = SACService.addAuthPlugin()
        pluginConfigID = response.getId()
        println response.toString()
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        if (pluginConfigID != null) {
            SACService.deleteAuthPluginConfig(pluginConfigID)
        }
    }

    @Unroll
    @Category([BAT])
    def "Verify that only namespace #sacResource is visible when using SAC"() {
        when:
        "Create test API token with a built-in role"
        GenerateTokenResponse token = ApiTokenService.
                generateToken("deployments-access-token", "None")
        BaseService.useApiToken(token.token)
        then:
        "Call API and verify data returned is within scoped access"
        def result = DeploymentService.listDeployments()
        println result.toString()
        assert result.size() == 1
        def resourceNotAllowed = result.find { it.namespace != sacResource }
        assert resourceNotAllowed == null
        cleanup:
        BaseService.useBasicAuth()
        where:
        "Data inputs are: "
        sacResource | _
        "test-qa2" | _
    }
}
