import static util.Helpers.withRetry

import orchestratormanager.OrchestratorTypes

import objects.Deployment
import objects.K8sServiceAccount
import objects.NetworkPolicyTypes
import objects.Service
import objects.Pagination
import services.NetworkGraphService
import util.CollectorUtil
import util.NetworkGraphUtil
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Stepwise
import spock.lang.Tag

@Stepwise
@Tag("PZ")
class ExternalIpFlowsTest extends BaseSpecification {

    static final private String EXTERNALDESTINATION = "external-destination-source"

    static final private int RETRY_COUNT = 5
    static final private int RETRY_INTERVAL = 30

    static final private String BERSERKER_SUBNET = "223.42.0.0/24"
    static final private String DEFAULT_QUERY = "Namespace:qa+Discovered External Source:true+External Source Address:${BERSERKER_SUBNET}"

    def buildSourceDeployments() {
        return [
            new Deployment()
                    .setName(EXTERNALDESTINATION)
                    .setImage("quay.io/rhacs-eng/qa:berserker-network-1.0-84-g5775ea7b69")
                    .addLabel("app", EXTERNALDESTINATION)
                    .setHostNetwork(true)
                    .setPrivilegedFlag(true)
        ]
    }

    private List<Deployment> deployments

    def createDeployments() {
        deployments = buildSourceDeployments()
        orchestrator.batchCreateDeployments(deployments)
        for (Deployment d : deployments) {
            assert Services.waitForDeployment(d)
        }
    }

    def setupSpec() {
        CollectorUtil.enableExternalIps(orchestrator)
        createDeployments()
    }

    def destroyDeployments() {
        for (Deployment deployment : deployments) {
            orchestrator.deleteDeployment(deployment)
        }
        for (Deployment deployment : deployments) {
            if (deployment.exposeAsService) {
                orchestrator.waitForServiceDeletion(new Service(deployment.name, deployment.namespace))
            }
        }
    }

    def cleanupSpec() {
        CollectorUtil.deleteRuntimeConfig(orchestrator)
        destroyDeployments()
    }

    @Tag("NetworkFlowVisualization")
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify external network flow metadata"() {
        when:

        def metadataResponse
        withRetry(RETRY_COUNT, RETRY_INTERVAL) {
            metadataResponse = NetworkGraphService.getExternalNetworkFlowsMetadata(
                DEFAULT_QUERY,
            )
            assert metadataResponse.getTotalEntities() == 2
        }

        then: "expect two discovered entities with one flow each"
            metadataResponse.getEntitiesList().each { entity ->
                assert entity.getFlowsCount() == 1
            }
    }

    @Tag("NetworkFlowVisualization")
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify external network flow metadata paginate"() {
        when:

        def entities
        withRetry(RETRY_COUNT, RETRY_INTERVAL) {
            def metadataResponse = NetworkGraphService.getExternalNetworkFlowsMetadata(
                DEFAULT_QUERY,
                new Pagination(1, 0),
            )
            entities = metadataResponse.getEntitiesList()
            assert entities.size() != 0
        }

        then: "expect one discovered entity with one flow"
            def entity = entities[0]

            assert entity.getFlowsCount() == 1
    }

    @Tag("NetworkFlowVisualization")
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify external network flow metadata by CIDR"() {
        when:
        "Deployment A, where A communicates to an external target"
        String deploymentUid = deployments.find { it.name == EXTERNALDESTINATION }?.deploymentUid
        assert deploymentUid != null

        and:
        "Check for external flow"

        def extIp
        // retrying the first case, to wait for the network flows to appear,
        // then subsequent test cases are querying the same data in different ways
        withRetry(RETRY_COUNT, RETRY_INTERVAL) {
            def response = NetworkGraphService.getExternalNetworkFlowsMetadata(
                "${DEFAULT_QUERY}+DeploymentName:${EXTERNALDESTINATION}"
            )

            assert response != null
            assert response.getEntitiesList()?.size() == 2

            // retrieve the IP. We can use this to filter later
            extIp = response.getEntities(0)?.getExternalSource()?.getCidr()
            assert extIp != null
        }

        // construct some subnets to use to verify filtering of
        // expected external entity
        def octets = extIp.tokenize(".")
        def widerNet = "${octets[0]}.${octets[1]}.0.0/16"
        def narrowerNet = "${octets[0]}.${octets[1]}.${octets[2]}.254/31"

        and: "Get no flows with CIDR filter"
        def cidrFilteredFlows = NetworkGraphService.getExternalNetworkFlowsMetadata(
            "${DEFAULT_QUERY}+External Source Address:123.123.123.0/24"
        )

        and: "Get flows with wide subnet filter"
        def widerNetFlows = NetworkGraphService.getExternalNetworkFlowsMetadata(
            "${DEFAULT_QUERY}+External Source Address:${widerNet}"
        )

        and: "Get flows with narrow subnet filter"
        def narrowerNetFlows = NetworkGraphService.getExternalNetworkFlowsMetadata(
            "${DEFAULT_QUERY}+External Source Address:${narrowerNet}"
        )

        and: "Get flows for non-existent namespace"
        def noNamespaceFlows = NetworkGraphService.getExternalNetworkFlowsMetadata(
            "Namespace:empty")

        then:
        assert cidrFilteredFlows?.getEntitiesList().size() == 0

        and:
        assert widerNetFlows?.getEntities().size() == 1
        assert widerNetFlows.getEntities(0)?.getExternalSource()?.getCidr() == extIp

        and:
        assert narrowerNetFlows?.getEntitiesList().size() == 0

        and:
        assert noNamespaceFlows?.getEntitiesList().size() == 0
    }
}
