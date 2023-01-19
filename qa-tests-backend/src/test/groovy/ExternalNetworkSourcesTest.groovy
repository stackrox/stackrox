import java.util.concurrent.TimeUnit

import com.google.protobuf.Timestamp

import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntity

import objects.Deployment
import objects.Edge
import services.ClusterService
import services.NetworkGraphService
import util.NetworkGraphUtil

import spock.lang.Tag

class ExternalNetworkSourcesTest extends BaseSpecification {
    // Any reliable static IP address should work here.
    // For now we use the one belonging to CloudFlare
    // in hopes it doesn't disappear.
    static final private String CF_IP_ADDRESS = "1.1.1.1"
    static final private String CF_CIDR_30 = "$CF_IP_ADDRESS/30"
    static final private String CF_CIDR_31 = "$CF_IP_ADDRESS/31"

    static final private String EXT_CONN_DEPLOYMENT_NAME = "external-connection"

    static final private List<Deployment> DEPLOYMENTS = []

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

    @Tag("NetworkFlowVisualization")
    def "Verify connection to a user created external sources"() {
        when:
        "Deployment is communicating with Cloudflare's IP address"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a external source containing Cloudflare's IP address"
        String externalSourceName = "external-source"
        NetworkEntity externalSource = createNetworkEntityExternalSource(externalSourceName, CF_CIDR_31)
        String externalSourceID = externalSource?.getInfo()?.getId()
        assert externalSourceID != null

        then:
        "Verify edge from deployment to external source exists"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(deploymentUid, externalSourceID, null, 150)
        assert edges

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
        String externalSource31Name = "external-source-31"
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, CF_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        log.info "Edge from deployment to external source ${externalSource31Name} should exist"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)

        log.info "Create a supernet external source containing Cloudflare's IP address"
        String externalSource30Name = "external-source-30"
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, CF_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30ID != null

        then:
        "Verify no edge from deployment to supernet exists"
        verifyNoEdge(deploymentUid, externalSource30ID, null)

        and:
        "Verify edge from deployment to subnet still exists"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)

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
        String externalSource31Name = "external-source-31"
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, CF_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        then:
        "Verify edge from deployment to subnet exists before subnet deletion"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)

        log.info "Add supernet and remove subnet"
        String externalSource30Name = "external-source-30"
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, CF_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30ID != null

        and:
        "Verify no edge from deployment to supernet exists before subnet deletion"
        verifyNoEdge(deploymentUid, externalSource30ID, null)

        "Remove the smaller subnet should add an edge to the larger subnet"
        deleteNetworkEntity(externalSource31ID)

        and:
        "Verify edge from deployment to supernet exists after subnet deletion"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource30ID, null, 180)

        cleanup:
        deleteNetworkEntity(externalSource30ID)
        deleteNetworkEntity(externalSource31ID)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify two flows co-exist if larger network entity added first"() {
        when:
        "Supernet external source is created before subnet external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        String externalSource30Name = "external-source-30"
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, CF_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30ID != null

        log.info "Verify edge exists from deployment to supernet external source"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource30ID)

        log.info "Add smaller subnet subnet external source"
        String externalSource31Name = "external-source-31"
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, CF_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        then:
        "Verify edge exists from deployment to subnet external source"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID, null, 180)

        and:
        "Verify edge from deployment to supernet exists in older network graph"
        assert NetworkGraphUtil.checkForEdge(
                deploymentUid,
                externalSource30ID,
                Timestamp.newBuilder().setSeconds(System.currentTimeSeconds() - 60*60).build())

        // Wait for some time for to accommodate delays in receiving flows from collector, etc.
        TimeUnit.SECONDS.sleep(NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS)

        and:
        "Verify no edge from deployment to supernet exists in recent network graph"
        assert verifyNoEdge(
                deploymentUid,
                externalSource30ID,
                Timestamp.newBuilder().setSeconds(System.currentTimeSeconds() - 60).build())

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
        NetworkGraphService.deleteNetworkEntity(entityID)
    }

    private verifyNoEdge(String entityID1, String entityID2, Timestamp since) {
        // First wait for some time to give time for network flow update
        log.info "Wait a bit before checking graph to verify no edge..."
        TimeUnit.SECONDS.sleep(NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS*2)
        // Shorter timeout for verifying no edge to save test time
        return !NetworkGraphUtil.checkForEdge(
                entityID1,
                entityID2,
                since,
                10)
    }
}
