import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.storage.DeploymentOuterClass

import objects.Deployment
import services.DeploymentService
import util.Env

import org.junit.Assume
import spock.lang.Tag

class RoutesTest extends BaseSpecification {

    static final private SERVER_DEP = new Deployment()
        .setName("server")
        .setImage("quay.io/rhacs-eng/qa:nginx-1.19-alpine")
        .addLabel("app", "server")
        .addPort(80)
        .setExposeAsService(true)

    def getRoutes() {
        def deployment = DeploymentService.getDeployment(SERVER_DEP.deploymentUid)
        def ports = deployment.getPortsList()
        assert ports.size() == 1
        def port = ports[0]
        port.getExposureInfosList().findAll { it.getLevel() == DeploymentOuterClass.PortConfig.ExposureLevel.ROUTE }
    }

    @Tag("BAT")
    def "Verify that routes are detected correctly"() {
        boolean routeCreated

        given:
        Assume.assumeTrue(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)

        when:
        "Create the deployment"
        orchestrator.createDeployment(SERVER_DEP)
        assert SERVER_DEP.deploymentUid

        then:
        "Fetch deployment, it shouldn't have a route"
        withRetry(10, 5) {
            def routes = getRoutes()
            assert routes.size() == 0
        }

        when:
        "Create a route"
        orchestrator.createRoute(SERVER_DEP.name, SERVER_DEP.namespace)
        routeCreated = true

        then:
        "Fetch deployment, it should have the route"
        withRetry(10, 5) {
            def routes = getRoutes()
            assert routes.size() == 1
            assert routes[0].getExternalHostnamesList().size() > 0
        }

        when:
        "Delete the route"
        orchestrator.deleteRoute(SERVER_DEP.name, SERVER_DEP.namespace)
        routeCreated = false

        then:
        "Fetch deployment, it should no longer have the route"
        withRetry(10, 5) {
            def routes = getRoutes()
            assert routes.size() == 0
        }

        cleanup:
        orchestrator.deleteDeployment(SERVER_DEP)
        if (routeCreated) {
            orchestrator.deleteRoute(SERVER_DEP.name, SERVER_DEP.namespace)
        }
    }
}
