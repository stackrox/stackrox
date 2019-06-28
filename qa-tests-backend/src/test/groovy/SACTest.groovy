import objects.Deployment
import services.SACService
import services.BaseService
import spock.lang.Shared

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
        BaseService.useBasicAuth()
    }
}
