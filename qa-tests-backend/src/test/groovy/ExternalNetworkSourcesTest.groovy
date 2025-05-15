import com.google.protobuf.Timestamp

import io.stackrox.annotations.Retry
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntity

import objects.Deployment
import services.ClusterService
import services.NetworkGraphService
import util.NetworkGraphUtil

import spock.lang.Shared
import spock.lang.Tag

@Tag("PZ")
class ExternalNetworkSourcesTest extends BaseSpecification {
    // Any reliable static IP address should work here.
    // For now we use the one belonging to CloudFlare
    // in hopes it doesn't disappear.
    static final private String CF_IP_ADDRESS = "1.1.1.1"

    static final private String EXT_CONN_DEPLOYMENT_NAME = "external-connection"

    static final private List<Deployment> DEPLOYMENTS = []

    static final private RANDOM = new Random()

    static final private Deployment DEP_EXTERNALCONNECTION =
            createAndRegisterDeployment()
                    .setName(EXT_CONN_DEPLOYMENT_NAME)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-19-alpine")
                    .addLabel("app", EXT_CONN_DEPLOYMENT_NAME)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS / 10}; " +
                                      "do wget -S ${CF_IP_ADDRESS}; " +
                                      "done" as String,])

    private static createAndRegisterDeployment() {
        Deployment deployment = new Deployment()
        DEPLOYMENTS.add(deployment)
        return deployment
    }

    @Shared
    private int mask = 0

    private String getSupernetCIDR() {
        return "$CF_IP_ADDRESS/${15 + mask}"
    }

    private String getSubnetCIDR() {
        return "$CF_IP_ADDRESS/${30 - mask}"
    }

    def setup() {
        mask++
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanup() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    private static String generateNameWithPrefix(String prefix) {
        var externalSourceId = RANDOM.nextInt() % 1000
        return "$prefix-$externalSourceId"
    }

    @Tag("NetworkFlowVisualization")
    def "Verify connection to a user created external sources"() {
        when:
        "Deployment is communicating with Cloudflare's IP address"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a external source containing Cloudflare's IP address"
        String externalSourceName = generateNameWithPrefix("external-source")
        NetworkEntity externalSource = createNetworkEntityExternalSource(externalSourceName, subnetCIDR)
        String externalSourceID = externalSource.getInfo().getId()

        then:
        "Verify edge from deployment to external source exists"
        verifyEdge(deploymentUid, externalSourceID)

        cleanup:
        "Remove the external source and associated deployments"
        deleteNetworkEntity(externalSourceID)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify flow stays to the smallest subnet possible"() {
        when:
        "Supernet external source is created after subnet external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a smaller network external source containing Cloudflare's IP address"
        String subnetName = generateNameWithPrefix("external-source-subnet")
        NetworkEntity subnet = createNetworkEntityExternalSource(subnetName, subnetCIDR)
        String subnetID = subnet.getInfo().getId()

        log.info "Edge from deployment to external source ${subnetName} should exist"
        verifyEdge(deploymentUid, subnetID)

        log.info "Create a supernet external source containing Cloudflare's IP address"
        String supernetName = generateNameWithPrefix("external-source-supernet")
        NetworkEntity supernet = createNetworkEntityExternalSource(supernetName, supernetCIDR)
        String supernetID = supernet.getInfo().getId()

        then:
        "Verify no edge from deployment to supernet exists"
        verifyNoEdge(deploymentUid, supernetID)

        and:
        "Verify edge from deployment to subnet still exists"
        verifyEdge(deploymentUid, subnetID)

        cleanup:
        deleteNetworkEntity(supernetID)
        deleteNetworkEntity(subnetID)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify flow re-maps to larger subnet when smaller subnet deleted"() {
        when:
        "Supernet is added after subnet followed by subnet deletion"

        log.info "Get ID of deployment which is communicating the external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a smaller subnet network entities with Cloudflare's IP address"
        String subnetName = generateNameWithPrefix("external-source-subnet")
        NetworkEntity subnet = createNetworkEntityExternalSource(subnetName, subnetCIDR)
        String subnetID = subnet.getInfo().getId()

        then:
        "Verify edge from deployment to subnet exists before subnet deletion"
        verifyEdge(deploymentUid, subnetID)

        log.info "Add supernet and remove subnet"
        String supernetName = generateNameWithPrefix("external-source-supernet")
        NetworkEntity supernet = createNetworkEntityExternalSource(supernetName, supernetCIDR)
        String supernetID = supernet.getInfo().getId()

        and:
        "Verify no edge from deployment to supernet exists before subnet deletion"
        verifyNoEdge(deploymentUid, supernetID)

        "Remove the smaller subnet should add an edge to the larger subnet"
        deleteNetworkEntity(subnetID)

        and:
        "Verify edge from deployment to supernet exists after subnet deletion"
        verifyEdge(deploymentUid, supernetID)

        cleanup:
        deleteNetworkEntity(supernetID)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify two flows co-exist if larger network entity added first"() {
        when:
        "Supernet external source is created before subnet external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        String supernetName = generateNameWithPrefix("external-source-supernet")
        NetworkEntity supernet = createNetworkEntityExternalSource(supernetName, supernetCIDR)
        String supernetID = supernet.getInfo().getId()

        log.info "Verify edge exists from deployment to supernet external source"
        verifyEdge(deploymentUid, supernetID)

        log.info "Add smaller subnet subnet external source"
        String subnetName = generateNameWithPrefix("external-source-subnet")
        NetworkEntity subnet = createNetworkEntityExternalSource(subnetName, subnetCIDR)
        String subnetID = subnet.getInfo().getId()

        then:
        "Verify edge exists from deployment to subnet external source"
        verifyEdge(deploymentUid, subnetID)

        and:
        "Verify edge from deployment to supernet exists in older network graph"
        verifyEdge(
                deploymentUid,
                supernetID,
                60 * 60)

        and:
        "Verify no edge from deployment to supernet exists in recent network graph"
        verifyNoEdge(
                deploymentUid,
                supernetID,
                60)

        cleanup:
        deleteNetworkEntity(supernetID)
        deleteNetworkEntity(subnetID)
    }

    private static NetworkEntity createNetworkEntityExternalSource(String name, String cidr) {
        String clusterId = ClusterService.getClusterId()
        NetworkGraphService.createNetworkEntity(clusterId, name, cidr, false)
        return NetworkGraphService.waitForNetworkEntityOfExternalSource(clusterId, name)
    }

    private static deleteNetworkEntity(String entityID) {
        // Use network graph client without the wrapper because we need the test to fail if the deletion fails.
        NetworkGraphService.getNetworkGraphClient()
                .deleteExternalNetworkEntity(Common.ResourceByID.newBuilder().setId(entityID).build())
    }

    @Retry(attempts = 4, delay = 30)
    private static void verifyEdge(String deploymentUid, String subnetID, int sinceSeconds = 0) {
        assert NetworkGraphUtil.checkForEdge(deploymentUid, subnetID, since(sinceSeconds), 180)
    }

    @Retry(attempts = 4, delay = 30)
    private static void verifyNoEdge(String entityID1, String entityID2, int sinceSeconds = 0) {
        assert !NetworkGraphUtil.checkForEdge(
                entityID1,
                entityID2,
                since(sinceSeconds),
                10)
    }

    private static Timestamp since(int sinceSeconds) {
        if (sinceSeconds <= 0) {
            return null
        }
        return Timestamp.newBuilder().setSeconds(System.currentTimeSeconds() - sinceSeconds).build()
    }
}
