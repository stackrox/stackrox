import static util.Helpers.withRetry

import com.google.common.collect.Sets
import com.google.protobuf.Timestamp

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass
import io.stackrox.proto.storage.NetworkFlowOuterClass
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
        Set<String> similarCIDRs = getAllCIDRs().findAll { it.startsWith("1.1.") }
        log.info("Post-cleanup state of similar CIDRs: ${similarCIDRs}")
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
        Set<String> similarCIDRs = getAllCIDRs().findAll { it.startsWith("1.1.") }
        log.info("Pre-case-cleanup state of similar CIDRs: ${similarCIDRs}")
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
        Set<String> similarCIDRs = getAllCIDRs().findAll { it.startsWith("1.1.") }
        log.info("Pre-case-cleanup state of similar CIDRs: ${similarCIDRs}")
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
        Set<String> similarCIDRs = getAllCIDRs().findAll { it.startsWith("1.1.") }
        log.info("Pre-case-cleanup state of similar CIDRs: ${similarCIDRs}")
        deleteNetworkEntity(externalSource30ID)
    }

    @Tag("NetworkFlowVisualization")
    /*
    This test case creates the "external-connection" deployment and two external sources representing
    an endpoint within CIDRs 1.1.1.1/30 and 1.1.1.1/31. The /30 is added first, then /31 follows.
    It ensures that network graph has an edge between the "external-connection" deployment and the /30 CIDR, and then
    between "external-connection" deployment and the /31 CIDR.
    Next, it looks only at the network graph generated for the last 60s and expects that the edge to /30 disappears,
    while the edge to /31 is still present.
    */
    def "Verify two flows co-exist if larger network entity added first"() {
        when:
        "Supernet external source is created before subnet external source"
        String deploymentUid = DEP_EXTERNALCONNECTION.deploymentUid
        assert deploymentUid != null

        Set<String> potentiallyConflictingCIDRs = [CF_CIDR_30, CF_CIDR_31, "1.1.1.0/30", "1.1.1.0/31"] as Set<String>
        Set<String> allCIDRs = getAllCIDRs()
        Set<String> similarCIDRs = allCIDRs.findAll { it.startsWith("1.1.") }
        Sets.SetView<String> conflictingCIDRs = Sets.intersection(potentiallyConflictingCIDRs, similarCIDRs)
        if (conflictingCIDRs.isEmpty()) {
            log.debug("Found no CIDRs conflicting with ${potentiallyConflictingCIDRs}." +
                "All custom CIDRs currently in Central: ${similarCIDRs}")
        } else {
            log.warn("found existing CIDR blocks ${conflictingCIDRs} that conflict with this test case." +
                "Check the cleanup of other tests if the interference causes this test to fail." +
                "All custom CIDRs currently in Central: ${similarCIDRs}")
        }

        getMatchingCIDRs(CF_CIDR_30).forEach { it ->
            log.warn("Deleting conflicting CIDR " +
                "id: ${it.getId()}, " +
                "name: ${it.getExternalSource().name}, " +
                "cidr: ${it.getExternalSource().cidr}")
            deleteNetworkEntity(it.getId())
        }
        log.info("All external entities conflicting with ${CF_CIDR_30} have been deleted")

        String externalSource30Name = generateNameWithPrefix("external-source-30")
        log.info("Creating external source '${externalSource30Name}' with CIDR ${CF_CIDR_30}")
        NetworkEntity externalSource30 = createNetworkEntityExternalSource(externalSource30Name, CF_CIDR_30)
        String externalSource30ID = externalSource30?.getInfo()?.getId()
        assert externalSource30 != null
        assert externalSource30ID != null

        log.info "Verify edge exists from deployment 'external-connection' to " +
            "supernet external source '$externalSource30Name'"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource30ID)
        }

        log.info "Add smaller subnet subnet external source"

        getMatchingCIDRs(CF_CIDR_31).forEach { it ->
            log.warn("Deleting conflicting CIDR " +
                "id: ${it.getId()}, " +
                "name: ${it.getExternalSource().name}, " +
                "cidr: ${it.getExternalSource().cidr}")
            deleteNetworkEntity(it.getId())
        }
        log.info("All external entities conflicting with ${CF_CIDR_31} have been deleted")

        String externalSource31Name = generateNameWithPrefix("external-source-31")
        NetworkEntity externalSource31 = createNetworkEntityExternalSource(externalSource31Name, CF_CIDR_31)
        String externalSource31ID = externalSource31?.getInfo()?.getId()
        assert externalSource31 != null
        assert externalSource31ID != null

        then:
        "Verify edge exists from deployment 'external-connection' to subnet external source '$externalSource31Name'"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(deploymentUid, externalSource31ID, null, 180)
        }

        and:
        "Verify edge from deployment 'external-connection' to supernet external source '$externalSource30Name' exists" +
            " in the network graph for last 60 minutes"
        withRetry(4, 30) {
            assert NetworkGraphUtil.checkForEdge(
                deploymentUid,
                externalSource30ID,
                Timestamp.newBuilder().setSeconds(System.currentTimeSeconds() - 60 * 60).build())
        }

        and:
        "Verify no edge exists from 'external-connection' to supernet external source '$externalSource30Name' " +
            "in the network graph for last 60 seconds"
        withRetry(20, 30) {
            // We need to wait for at least (i.e., the sum of all the following):
            // - another enrichment to happen - at least 30s.
            // - old connection being closed by Collector - at least 5 min with the default afterglow setting.
            // - the time-scope of the network graph - here 60s.
            // Waiting for 6 min and 30s is a minimum here, however the edge may disappear sooner.
            // Manually observing this test-case confirmed that there are two updates for "externalSource30ID"
            // sent to Central, the second exactly 5min30s after the first one.
            // We set the retries to cover 10 minutes to account for unpredictable issues.
            assert verifyNoEdge(
                deploymentUid,
                externalSource30ID,
                Timestamp.newBuilder().setSeconds(System.currentTimeSeconds() - 60).build())
        }

        cleanup:
        Set<String> similarCIDRs2 = getAllCIDRs().findAll { it.startsWith("1.1.") }
        log.info("Pre-case-cleanup state of similar CIDRs: ${similarCIDRs2}")
        deleteNetworkEntity(externalSource30ID)
        deleteNetworkEntity(externalSource31ID)
    }

    private static Set<String> getAllCIDRs() {
        def clusterId = ClusterService.getClusterId()
        assert clusterId
        def request = NetworkGraphServiceOuterClass.GetExternalNetworkEntitiesRequest
            .newBuilder()
            .setClusterId(clusterId)
            .build()
        def response = NetworkGraphService.getNetworkGraphClient().getExternalNetworkEntities(request)

        Set<String> existingCidrs = response.getEntitiesList().findAll {
            it.getInfo().hasExternalSource()
        }.collect {
            it.getInfo().getExternalSource().cidr
        } as Set

        return existingCidrs
    }

    private static Set<NetworkFlowOuterClass.NetworkEntityInfo> getMatchingCIDRs(String prefix) {
        def clusterId = ClusterService.getClusterId()
        assert clusterId
        def request = NetworkGraphServiceOuterClass.GetExternalNetworkEntitiesRequest
            .newBuilder()
            .setClusterId(clusterId)
            .build()
        def response = NetworkGraphService.getNetworkGraphClient().getExternalNetworkEntities(request)

        return response.getEntitiesList().findAll({
            it.getInfo().hasExternalSource() && it.getInfo().getExternalSource().cidr.startsWith(prefix)
        })*.getInfo() as Set
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
