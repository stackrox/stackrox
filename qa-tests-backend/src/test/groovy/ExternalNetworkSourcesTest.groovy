import static util.Helpers.withRetry

import com.google.protobuf.Timestamp

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntity

import objects.Deployment
import services.ClusterService
import services.NetworkGraphService
import util.NetworkGraphUtil

import spock.lang.Tag

@Tag("PZ")
class ExternalNetworkSourcesTest extends BaseSpecification {
    // Any reliable static IP address should work here.
    // For now we use the one belonging to CloudFlare
    // in hopes it doesn't disappear.
    static final private String CF_IP_ADDRESS = "1.1.1.1"
    static final private String CF_CIDR_30 = "$CF_IP_ADDRESS/30"
    static final private String CF_CIDR_31 = "$CF_IP_ADDRESS/31"

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

    def setup() {
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
        NetworkEntity externalSource = createNetworkEntityExternalSource(externalSourceName, CF_CIDR_31)
        String externalSourceID = externalSource?.getInfo()?.getId()
        assert externalSourceID != null

        then:
        "Verify edge from deployment to external source exists"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSourceID, null, 150)
        }

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
        String externalSource31Name = generateNameWithPrefix("external-source-31")
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, CF_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        log.info "Edge from deployment to external source ${externalSource31Name} should exist"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)
        }

        log.info "Create a supernet external source containing Cloudflare's IP address"
        String externalSource30Name = generateNameWithPrefix("external-source-30")
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, CF_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30ID != null

        then:
        "Verify no edge from deployment to supernet exists"
        withRetry(4, 30) {
            verifyNoEdge(deploymentUid, externalSource30ID, null)
        }

        and:
        "Verify edge from deployment to subnet still exists"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)
        }

        cleanup:
        deleteNetworkEntity(externalSource30ID)
        deleteNetworkEntity(externalSource31ID)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify flow re-maps to larger subnet when smaller subnet deleted"() {
        when:
        "Supernet is added after subnet followed by subnet deletion"

        log.info "Get ID of deployment which is communicating the external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a smaller subnet network entities with Cloudflare's IP address"
        String externalSource31Name = generateNameWithPrefix("external-source-31")
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, CF_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        then:
        "Verify edge from deployment to subnet exists before subnet deletion"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)
        }

        log.info "Add supernet and remove subnet"
        String externalSource30Name = generateNameWithPrefix("external-source-30")
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, CF_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30ID != null

        and:
        "Verify no edge from deployment to supernet exists before subnet deletion"
        withRetry(4, 30) {
            verifyNoEdge(deploymentUid, externalSource30ID, null)
        }

        "Remove the smaller subnet should add an edge to the larger subnet"
        deleteNetworkEntity(externalSource31ID)

        and:
        "Verify edge from deployment to supernet exists after subnet deletion"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource30ID, null, 180)
        }

        cleanup:
        deleteNetworkEntity(externalSource30ID)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify two flows co-exist if larger network entity added first"() {
        when:
        "Supernet external source is created before subnet external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        String externalSource30Name = generateNameWithPrefix("external-source-30")
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, CF_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30ID != null

        log.info "Verify edge exists from deployment to supernet external source"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource30ID)
        }

        log.info "Add smaller subnet subnet external source"
        String externalSource31Name = generateNameWithPrefix("external-source-31")
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, CF_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        then:
        "Verify edge exists from deployment to subnet external source"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID, null, 180)
        }

        and:
        "Verify edge from deployment to supernet exists in older network graph"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(
                    deploymentUid,
                    externalSource30ID,
                    Timestamp.newBuilder().setSeconds(System.currentTimeSeconds() - 60*60).build())
        }

        and:
        "Verify no edge from deployment to supernet exists in recent network graph"
        withRetry(4, 30) {
            assert verifyNoEdge(
                    deploymentUid,
                    externalSource30ID,
                    Timestamp.newBuilder().setSeconds(System.currentTimeSeconds() - 60).build())
        }

        cleanup:
        deleteNetworkEntity(externalSource30ID)
        deleteNetworkEntity(externalSource31ID)
    }

    private static createNetworkEntityExternalSource(String name, String cidr) {
        String clusterId = ClusterService.getClusterId()
        NetworkGraphService.createNetworkEntity(clusterId, name, cidr, false)
        return NetworkGraphService.waitForNetworkEntityOfExternalSource(clusterId, name)
    }

    private static deleteNetworkEntity(String entityID) {
        // Use network graph client without the wrapper because we need the test to fail if the deletion fails.
        NetworkGraphService.getNetworkGraphClient()
            .deleteExternalNetworkEntity(Common.ResourceByID.newBuilder().setId(entityID).build())
    }

    private verifyNoEdge(String entityID1, String entityID2, Timestamp since) {
        // Shorter timeout for verifying no edge to save test time
        return !NetworkGraphUtil.checkForEdge(
                entityID1,
                entityID2,
                since,
                10)
    }
}
