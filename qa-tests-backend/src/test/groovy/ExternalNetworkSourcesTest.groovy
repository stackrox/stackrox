import groups.NetworkFlowVisualization
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntity
import objects.Deployment
import objects.Edge
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.FeatureFlagService
import services.NetworkGraphService
import util.NetworkGraphUtil

import java.util.concurrent.TimeUnit

class ExternalNetworkSourcesTest extends BaseSpecification {

    static final private String EXTERNAL_SOURCES_FEATURE_FLAG = "ROX_NETWORK_GRAPH_EXTERNAL_SRCS"

    // One of the outputs of: dig storage.googleapis.com
    // As of now it is stable, but the request would return 404 since the address
    // is used for resolving bucket names, but we are not supplying any bucket info.
    // TODO: potentially find a very reliable static IP that can be used for our testing
    static final private String GOOGLE_IP_ADDRESS = "172.217.6.48"
    static final private String GOOGLE_CIDR_8 = "172.0.0.0/8"
    static final private String GOOGLE_CIDR_16 = "172.217.0.0/16"

    static final private String EXTERNALCONNECTION = "external-connection"

    static final private List<Deployment> DEPLOYMENTS = []

    static final private Deployment DEP_EXTERNALCONNECTION =
            createAndRegisterDeployment()
                    .setName(EXTERNALCONNECTION)
                    .setImage("nginx:1.19-alpine")
                    .addLabel("app", EXTERNALCONNECTION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS / 10}; " +
                                      "do wget -S ${GOOGLE_IP_ADDRESS}; " +
                                      "done" as String,])

    private static createAndRegisterDeployment() {
        Deployment deployment = new Deployment()
        DEPLOYMENTS.add(deployment)
        return deployment
    }

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Category([NetworkFlowVisualization])
    def "Verify connection to a user created external sources"() {
        setup:
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled(EXTERNAL_SOURCES_FEATURE_FLAG))

        "Create a network entity with Google's IP address"
        String externalSourceName = "external-source"
        NetworkEntity externalSource = createNetworkEntityExternalSource(externalSourceName, GOOGLE_CIDR_8)
        String externalSourceID = externalSource?.getInfo()?.getId()
        assert externalSourceID != null

        "Get ID of deployment which is communicating the external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        expect:
        "Check for edge in network graph"
        println "Checking for edge from ${EXTERNALCONNECTION} to external network entity ${externalSourceName}"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(deploymentUid, externalSourceID, null, 150)
        assert edges

        cleanup:
        "Remove the network entity"
        deleteNetworkEntity(externalSourceID)
    }

    @Category([NetworkFlowVisualization])
    def "Verify flow stays to the smallest subnet possible"() {
        setup:
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled(EXTERNAL_SOURCES_FEATURE_FLAG))
        // Skipping for now since bugs
        def skip = true
        Assume.assumeFalse(skip)

        "Create a smaller subnet network entity with Google's IP address"
        String externalSource16Name = "external-source-16"
        NetworkEntity externalSource16 = createNetworkEntityExternalSource(externalSource16Name, GOOGLE_CIDR_16)
        String externalSource16ID = externalSource16?.getInfo()?.getId()
        assert externalSource16ID != null

        "Get ID of deployment which is communicating the external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        expect:
        println "Checking for edge from ${EXTERNALCONNECTION} to external network entity ${externalSource16Name}"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource16ID)

        "Adding a larger subnet entity should not result in a new edge"
        String externalSource8Name = "external-source-8"
        NetworkEntity externalSource8 = createNetworkEntityExternalSource(externalSource8Name, GOOGLE_CIDR_8)
        String externalSource8ID = externalSource8?.getInfo().getId()
        assert externalSource8ID != null

        println "Checking NO edge from ${EXTERNALCONNECTION} to external network entity ${externalSource8Name}"
        verifyNoEdge(deploymentUid, externalSource8ID)

        println "Checking edge is still there from ${EXTERNALCONNECTION} " +
                "to external network entity ${externalSource16Name}"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource16ID)

        cleanup:
        deleteNetworkEntity(externalSource8ID)
        deleteNetworkEntity(externalSource16ID)
    }

    @Category([NetworkFlowVisualization])
    def "Verify flow re-maps to larger subnet when smaller subnet deleted"() {
        setup:
        // Skipping for now since bugs
        def skip = true
        Assume.assumeFalse(skip)

        "Create a smaller subnet network entities with Google's IP address"
        String externalSource16Name = "external-source-16"
        NetworkEntity externalSource16 = createNetworkEntityExternalSource(externalSource16Name, GOOGLE_CIDR_16)
        String externalSource16ID = externalSource16?.getInfo()?.getId()
        assert externalSource16ID != null

        "Get ID of deployment which is communicating the external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        expect:
        println "Checking for edge from ${EXTERNALCONNECTION} to external network entity ${externalSource16Name}"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource16ID)

        println "Add larger subnet and removing this smaller subnet"
        String externalSource8Name = "external-source-8"
        NetworkEntity externalSource8 = createNetworkEntityExternalSource(externalSource8Name, GOOGLE_CIDR_8)
        String externalSource8ID = externalSource8?.getInfo().getId()
        assert externalSource8ID != null

        println "Checking NO edge from ${EXTERNALCONNECTION} to external network entity ${externalSource8Name}"
        verifyNoEdge(deploymentUid, externalSource8ID)

        "Remove the smaller subnet should add an edge to the larger subnet"
        deleteNetworkEntity(externalSource16ID)
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource8ID, null, 180)

        cleanup:
        // Need to delete both in case test fails midway
        deleteNetworkEntity(externalSource8ID)
        deleteNetworkEntity(externalSource16ID)
    }

    @Category([NetworkFlowVisualization])
    def "Verify two flows co-exist if larger network entity added first"() {
        setup:
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled(EXTERNAL_SOURCES_FEATURE_FLAG))
        // Skipping for now since bugs
        def skip = true
        Assume.assumeFalse(skip)

        "Create a larger subnet network entity with Google's IP address"
        String externalSource8Name = "external-source-8"
        NetworkEntity externalSource8 = createNetworkEntityExternalSource(externalSource8Name, GOOGLE_CIDR_8)
        String externalSource8ID = externalSource8?.getInfo()?.getId()
        assert externalSource8ID != null

        "Get ID of deployment which is communicating the external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        expect:
        println "Checking for edge from ${EXTERNALCONNECTION} to external network entity ${externalSource8Name}"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource8ID)

        "Adding a smaller subnet entity should add another edge, and old edge should also be retained"
        String externalSource16Name = "external-source-16"
        NetworkEntity externalSource16 = createNetworkEntityExternalSource(externalSource16Name, GOOGLE_CIDR_16)
        String externalSource16ID = externalSource16?.getInfo().getId()
        assert externalSource16ID != null

        println "Checking for edge from ${EXTERNALCONNECTION} to external network entity ${externalSource16Name}"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource16ID, null, 180)
        println "Checking edge is still there from ${EXTERNALCONNECTION} " +
                "to external network entity ${externalSource8Name}"
        assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource8ID)

        cleanup:
        deleteNetworkEntity(externalSource8ID)
        deleteNetworkEntity(externalSource16ID)
    }

    private static createNetworkEntityExternalSource(String name, String cidr) {
        String clusterId = ClusterService.getClusterId()
        NetworkGraphService.createNetworkEntity(clusterId, name, cidr, false)
        return NetworkGraphService.waitForNetworkEntityOfExternalSource(clusterId, name)
    }

    private static deleteNetworkEntity(String entityID) {
        NetworkGraphService.deleteNetworkEntity(entityID)
    }

    private static verifyNoEdge(String entityID1, String entityID2) {
        // First wait for some time to give time for network flow update
        println "Wait a bit before checking graph to verify no edge..."
        TimeUnit.SECONDS.sleep(NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS*2)
        // Shorter timeout for verifying no edge to save test time
        return !NetworkGraphUtil.checkForEdge(
                entityID1,
                entityID2,
                null,
                10)
    }
}
