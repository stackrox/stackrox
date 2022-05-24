import com.google.protobuf.Timestamp
import groups.NetworkFlowVisualization
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntity
import java.util.concurrent.TimeUnit
import objects.Deployment
import objects.Edge
import org.junit.experimental.categories.Category
import services.ClusterService
import services.NetworkGraphService
import util.NetworkGraphUtil

class ExternalNetworkSourcesTest extends BaseSpecification {
    // One of the outputs of: dig storage.googleapis.com
    // As of now it is stable, but the request would return 404 since the address
    // is used for resolving bucket names, but we are not supplying any bucket info.
    // TODO: potentially find a very reliable static IP that can be used for our testing
    static final private String GOOGLE_IP_ADDRESS = "172.217.6.48"
    static final private String GOOGLE_CIDR_30 = "172.217.6.48/30"
    static final private String GOOGLE_CIDR_31 = "172.217.6.48/31"

    static final private String EXT_CONN_DEPLOYMENT_NAME = "external-connection"

    static final private List<Deployment> DEPLOYMENTS = []

    static final private Deployment DEP_EXTERNALCONNECTION =
            createAndRegisterDeployment()
                    .setName(EXT_CONN_DEPLOYMENT_NAME)
                    .setImage("quay.io/rhacs-eng/qa:nginx-1.19-alpine")
                    .addLabel("app", EXT_CONN_DEPLOYMENT_NAME)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS / 10}; " +
                                      "do wget -S ${GOOGLE_IP_ADDRESS}; " +
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

    @Category([NetworkFlowVisualization])
    def "Verify connection to a user created external sources"() {
        when:
        "Deployment is communicating with Google's IP address"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a external source containing Google's IP address"
        String externalSourceName = "external-source"
        NetworkEntity externalSource = createNetworkEntityExternalSource(externalSourceName, GOOGLE_CIDR_31)
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

    @Category([NetworkFlowVisualization])
    def "Verify flow stays to the smallest subnet possible"() {
        when:
        "Supernet external source is created after subnet external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a smaller network external source containing Google's IP address"
        String externalSource31Name = "external-source-31"
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, GOOGLE_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        log.info "Edge from deployment to external source ${externalSource31Name} should exist"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)

        log.info "Create a supernet external source containing Google's IP address"
        String externalSource30Name = "external-source-30"
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, GOOGLE_CIDR_30)
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

    @Category([NetworkFlowVisualization])
    def "Verify flow re-maps to larger subnet when smaller subnet deleted"() {
        when:
        "Supernet is added after subnet followed by subnet deletion"

        log.info "Get ID of deployment which is communicating the external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        log.info "Create a smaller subnet network entities with Google's IP address"
        String externalSource31Name = "external-source-31"
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, GOOGLE_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31ID != null

        then:
        "Verify edge from deployment to subnet exists before subnet deletion"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID)

        log.info "Add supernet and remove subnet"
        String externalSource30Name = "external-source-30"
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, GOOGLE_CIDR_30)
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

    @Category([NetworkFlowVisualization])
    def "Verify two flows co-exist if larger network entity added first"() {
        when:
        "Supernet external source is created before subnet external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        String externalSource30Name = "external-source-30"
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, GOOGLE_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30ID != null

        log.info "Verify edge exists from deployment to supernet external source"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource30ID)

        log.info "Add smaller subnet subnet external source"
        String externalSource31Name = "external-source-31"
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, GOOGLE_CIDR_31)
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
