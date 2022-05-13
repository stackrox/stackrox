import groups.NetworkBaseline
import io.stackrox.proto.storage.NetworkBaselineOuterClass
import io.stackrox.proto.storage.NetworkFlowOuterClass
import objects.Deployment
import org.junit.experimental.categories.Category
import services.NetworkBaselineService
import spock.lang.Ignore
import spock.lang.Retry
import util.NetworkGraphUtil

@Retry(count = 0)
class NetworkBaselineTest extends BaseSpecification {
    static final private String SERVER_DEP_NAME = "net-bl-server"
    static final private String BASELINED_CLIENT_DEP_NAME = "net-bl-client-baselined"
    static final private String ANOMALOUS_CLIENT_DEP_NAME = "net-bl-client-anomalous"
    static final private String DEFERRED_BASELINED_CLIENT_DEP_NAME = "net-bl-client-deferred-baselined"

    static final private String NGINX_IMAGE = "quay.io/rhacs-eng/qa:nginx-1.19-alpine"

    // The baseline generation duration must be changed from the default for this test to succeed.
    static final private int EXPECTED_BASELINE_DURATION_SECONDS = 120

    static final private int CLOCK_SKEW_ALLOWANCE_SECONDS = 15

    static final private List<Deployment> DEPLOYMENTS = []

    static final private SERVER_DEP = createAndRegisterDeployment()
                    .setName(SERVER_DEP_NAME)
                    .setImage(NGINX_IMAGE)
                    .addLabel("app", SERVER_DEP_NAME)
                    .addPort(80)
                    .setExposeAsService(true)

    static final private BASELINED_CLIENT_DEP = createAndRegisterDeployment()
                .setName(BASELINED_CLIENT_DEP_NAME)
                .setImage(NGINX_IMAGE)
                .addLabel("app", BASELINED_CLIENT_DEP_NAME)
                .setCommand(["/bin/sh", "-c",])
                .setArgs(
                    ["for i in \$(seq 1 10); do wget -S http://${SERVER_DEP_NAME}; sleep 1; done; sleep 1000" as String]
                )

    static final private ANOMALOUS_CLIENT_DEP = createAndRegisterDeployment()
        .setName(ANOMALOUS_CLIENT_DEP_NAME)
        .setImage(NGINX_IMAGE)
        .addLabel("app", ANOMALOUS_CLIENT_DEP_NAME)
        .setCommand(["/bin/sh", "-c",])
        .setArgs(["echo sleeping; sleep ${EXPECTED_BASELINE_DURATION_SECONDS+30}; echo sleep done; " +
                      "for i in \$(seq 1 10); do wget -S http://${SERVER_DEP_NAME}; sleep 1; done;" +
                      "sleep 1000" as String,])

    static final private DEFERRED_BASELINED_CLIENT_DEP = createAndRegisterDeployment()
        .setName(DEFERRED_BASELINED_CLIENT_DEP_NAME)
        .setImage(NGINX_IMAGE)
        .addLabel("app", DEFERRED_BASELINED_CLIENT_DEP_NAME)
        .setCommand(["/bin/sh", "-c",])
        .setArgs(["while sleep 1; " +
                      "do wget -S http://${SERVER_DEP_NAME}; " +
                      "done" as String,])

    private static createAndRegisterDeployment() {
        Deployment deployment = new Deployment()
        DEPLOYMENTS.add(deployment)
        return deployment
    }

    private batchCreate(List<Deployment> deployments) {
        orchestrator.batchCreateDeployments(deployments)
        for (Deployment deployment : deployments) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def validateBaseline(NetworkBaselineOuterClass.NetworkBaseline baseline, long beforeCreate,
                         long justAfterCreate, List<Tuple2<String, Boolean>> expectedPeers) {
        assert baseline.getObservationPeriodEnd().getSeconds() > beforeCreate - CLOCK_SKEW_ALLOWANCE_SECONDS
        assert baseline.getObservationPeriodEnd().getSeconds() <
            justAfterCreate + EXPECTED_BASELINE_DURATION_SECONDS + CLOCK_SKEW_ALLOWANCE_SECONDS
        assert baseline.getPeersCount() == expectedPeers.size()
        assert baseline.getForbiddenPeersCount() == 0

        for (def i = 0; i < expectedPeers.size(); i++) {
            def expectedPeerID = expectedPeers.get(i).getFirst()
            def expectedPeerIngress = expectedPeers.get(i).getSecond()
            def actualPeer = baseline.getPeersList().find { it.getEntity().getInfo().getId() == expectedPeerID }
            assert actualPeer
            def entityInfo = actualPeer.getEntity().getInfo()
            assert entityInfo.getType() == NetworkFlowOuterClass.NetworkEntityInfo.Type.DEPLOYMENT
            assert entityInfo.getId() == expectedPeerID
            assert actualPeer.getPropertiesCount() == 1
            def properties = actualPeer.getProperties(0)
            assert properties.getIngress() == expectedPeerIngress
            assert properties.getPort() == 80
            assert properties.getProtocol() == NetworkFlowOuterClass.L4Protocol.L4_PROTOCOL_TCP
        }
        return true
    }

    def cleanup() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Category(NetworkBaseline)
    @Ignore("Skip test for now, we're working on fixing it.")
    def "Verify network baseline functionality"() {
        when:
        "Create initial set of deployments, wait for baseline to populate"
        def beforeDeploymentCreate = System.currentTimeSeconds()
        batchCreate([SERVER_DEP, BASELINED_CLIENT_DEP, ANOMALOUS_CLIENT_DEP])
        def justAfterDeploymentCreate = System.currentTimeSeconds()

        def serverDeploymentID = SERVER_DEP.deploymentUid
        assert serverDeploymentID != null

        def baselinedClientDeploymentID = BASELINED_CLIENT_DEP.deploymentUid
        assert baselinedClientDeploymentID != null

        def anomalousClientDeploymentID = ANOMALOUS_CLIENT_DEP.deploymentUid
        assert anomalousClientDeploymentID != null
        log.info "Deployment IDs Server: ${serverDeploymentID}, " +
            "Baselined client: ${baselinedClientDeploymentID}, Anomalous client: ${anomalousClientDeploymentID}"

        assert NetworkGraphUtil.checkForEdge(baselinedClientDeploymentID, serverDeploymentID, null, 180)
        assert NetworkGraphUtil.checkForEdge(anomalousClientDeploymentID, serverDeploymentID, null,
            EXPECTED_BASELINE_DURATION_SECONDS + 180)

        def serverBaseline = evaluateWithRetry(20, 3) {
            def baseline = NetworkBaselineService.getNetworkBaseline(serverDeploymentID)
            if (baseline.getPeersCount() == 0) {
                throw new RuntimeException(
                    "No peers in baseline for deployment ${serverDeploymentID} yet. Baseline is ${baseline}"
                )
            }
            return baseline
        }
        assert serverBaseline
        def anomalousClientBaseline = NetworkBaselineService.getNetworkBaseline(anomalousClientDeploymentID)
        assert anomalousClientBaseline
        def baselinedClientBaseline = NetworkBaselineService.getNetworkBaseline(baselinedClientDeploymentID)
        assert baselinedClientDeploymentID

        then:
        "Validate server baseline"
        // The anomalous client->server connection should not be baselined since the anonymous client
        // sleeps for a time period longer than the observation period before connecting to the server.
        validateBaseline(serverBaseline, beforeDeploymentCreate, justAfterDeploymentCreate,
            [new Tuple2<String, Boolean>(baselinedClientDeploymentID, true)])
        validateBaseline(anomalousClientBaseline, beforeDeploymentCreate, justAfterDeploymentCreate, [])
        validateBaseline(baselinedClientBaseline, beforeDeploymentCreate, justAfterDeploymentCreate,
            [new Tuple2<String, Boolean>(serverDeploymentID, false)]
        )

        when:
        "Create another deployment, ensure it gets baselined"
        def beforeDeferredCreate = System.currentTimeSeconds()
        batchCreate([DEFERRED_BASELINED_CLIENT_DEP])
        def justAfterDeferredCreate = System.currentTimeSeconds()

        def deferredBaselinedClientDeploymentID = DEFERRED_BASELINED_CLIENT_DEP.deploymentUid
        assert deferredBaselinedClientDeploymentID != null

        assert NetworkGraphUtil.checkForEdge(anomalousClientDeploymentID, serverDeploymentID, null, 180)
        serverBaseline = evaluateWithRetry(20, 3) {
            def baseline = NetworkBaselineService.getNetworkBaseline(serverDeploymentID)
            if (baseline.getPeersCount() < 2) {
                throw new RuntimeException(
                    "Not enough peers in baseline for deployment ${serverDeploymentID} yet. Baseline is ${baseline}"
                )
            }
            return baseline
        }
        assert serverBaseline

        def deferredBaselinedClientBaseline = NetworkBaselineService.getNetworkBaseline(
            deferredBaselinedClientDeploymentID
        )
        assert deferredBaselinedClientDeploymentID

        then:
        "Validate the updated baselines"
        validateBaseline(serverBaseline, beforeDeploymentCreate, justAfterDeploymentCreate,
            [new Tuple2<String, Boolean>(baselinedClientDeploymentID, true),
             // Currently, we add conns to the baseline if it's within the observation period
             // of _at least_ one of the deployments. Therefore, the deferred client->server connection
             // gets added since it's within the deferred client's obervation period, and
             // the server's baseline is modified as well since we keep things consistent.
             new Tuple2<String, Boolean>(deferredBaselinedClientDeploymentID, true),
            ]
        )
        validateBaseline(deferredBaselinedClientBaseline, beforeDeferredCreate, justAfterDeferredCreate,
            [new Tuple2<String, Boolean>(serverDeploymentID, false)])
    }
}
