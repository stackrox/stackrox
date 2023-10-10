import static util.Helpers.withRetry

import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.storage.DeploymentOuterClass

import objects.Deployment
import services.DeploymentService
import util.Env

import org.junit.Assume
import spock.lang.Tag

class RoutesTest extends BaseSpecification {

    def getRoutes(String uuid) {
        def deployment = DeploymentService.getDeployment(uuid)
        def ports = deployment.getPortsList()
        assert ports.size() == 1
        def port = ports[0]
        port.getExposureInfosList().findAll { it.getLevel() == DeploymentOuterClass.PortConfig.ExposureLevel.ROUTE }
    }

    @Tag("BAT")
    @Tag("PZ")
    def "Verify that routes are detected correctly"() {
        given:
        Assume.assumeTrue(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)

        boolean resourcesCreated = false
        when:
        "Create the deployment"
        Deployment deployment = new Deployment()
                .setName(deploymentName)
                .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-19-alpine")
                .addLabel("app", deploymentName)
                .addPort(80)
                .setExposeAsService(autoExposeAsService)
        if (deployment.getExposeAsService()) {
            deployment.setCreateLoadBalancer(loadBalancer)
        }
        orchestrator.createDeployment(deployment)
        assert deployment.deploymentUid

        then:
        "Fetch deployment, it shouldn't have a route"
        withRetry(10, 5) {
            def routes = getRoutes(deployment.getDeploymentUid())
            assert routes.size() == 0
        }

        when:
        if (!autoExposeAsService && exposeAsService) {
            "Create a service"
            orchestrator.createService(deployment)
        }
        "Create a route"
        orchestrator.createRoute(deployment.name, deployment.namespace)
        resourcesCreated = true

        then:
        withRetry(14, 5) {
            def routes = getRoutes(deployment.getDeploymentUid())
            if (exposeAsService) {
                "Fetch deployment, it should have the route"
                assert routes.size() == 1
                assert routes[0].getExternalHostnamesList().size() > 0
            } else {
                "Fetch deployment, it should not have the route"
                assert routes.size() == 0
            }
        }

        when:
        if (!autoExposeAsService && exposeAsService) {
            "Delete the service"
            orchestrator.deleteService(deployment.name, deployment.namespace)
        }
        "Delete the route"
        orchestrator.deleteRoute(deployment.name, deployment.namespace)
        resourcesCreated = false

        then:
        "Fetch deployment, it should no longer have the route"
        withRetry(14, 5) {
            def routes = getRoutes(deployment.getDeploymentUid())
            assert routes.size() == 0
        }

        cleanup:
        orchestrator.deleteDeployment(deployment)
        if (resourcesCreated) {
            orchestrator.deleteRoute(deployment.name, deployment.namespace)
            if (!autoExposeAsService && exposeAsService) {
                orchestrator.deleteService(deployment.name, deployment.namespace)
            }
        }

        where:
        "Data is:"

        deploymentName                    | autoExposeAsService | exposeAsService | loadBalancer
        "server-auto-exposure-cluster-ip" | true                | true            | false
        "server-auto-exposure-lb"         | true                | true            |
                (!(Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x"))
        "server-exposure-cluster-ip"      | false               | true            | false
        "server-exposure-lb"              | false               | true            | true
        "server-no-exposure"              | false               | false           | false
    }
}
