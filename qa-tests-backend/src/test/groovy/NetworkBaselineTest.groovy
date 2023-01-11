import com.google.protobuf.Timestamp

import io.stackrox.proto.storage.NetworkBaselineOuterClass
import io.stackrox.proto.storage.NetworkFlowOuterClass

import objects.Deployment
import services.NetworkBaselineService
import util.NetworkGraphUtil
import util.Timer

import spock.lang.Retry
import spock.lang.Tag
import spock.lang.Timeout

@Retry(count = 0)
class NetworkBaselineTest extends BaseSpecification {
    static final private String SERVER_DEP_NAME = "net-bl-server"
    static final private String BASELINED_CLIENT_DEP_NAME = "net-bl-client-baselined"
    static final private String USER_DEP_NAME = "net-bl-user-server"
    static final private String BASELINED_USER_CLIENT_DEP_NAME = "net-bl-user-client-baselined"
    static final private String ANOMALOUS_CLIENT_DEP_NAME = "net-bl-client-anomalous"
    static final private String DEFERRED_BASELINED_CLIENT_DEP_NAME = "net-bl-client-deferred-baselined"
    static final private String DEFERRED_POST_LOCK_DEP_NAME = "net-bl-client-post-lock"

    static final private String NGINX_IMAGE = "quay.io/rhacs-eng/qa-multi-arch:nginx-1-19-alpine"

    // The baseline generation duration must be changed from the default for this test to succeed.
    static final private int EXPECTED_BASELINE_DURATION_SECONDS = 240

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

    static final private USER_DEP = createAndRegisterDeployment()
                    .setName(USER_DEP_NAME)
                    .setImage(NGINX_IMAGE)
                    .addLabel("app", USER_DEP_NAME)
                    .addPort(80)
                    .setExposeAsService(true)

    static final private BASELINED_USER_CLIENT_DEP = createAndRegisterDeployment()
                .setName(BASELINED_USER_CLIENT_DEP_NAME)
                .setImage(NGINX_IMAGE)
                .addLabel("app", BASELINED_USER_CLIENT_DEP_NAME)
                .setCommand(["/bin/sh", "-c",])
                .setArgs(["for i in \$(seq 1 10); do wget -S http://${USER_DEP_NAME};" +
                    "sleep 1; done; sleep 1000" as String])

    static final private ANOMALOUS_CLIENT_DEP = createAndRegisterDeployment()
        .setName(ANOMALOUS_CLIENT_DEP_NAME)
        .setImage(NGINX_IMAGE)
        .addLabel("app", ANOMALOUS_CLIENT_DEP_NAME)
        .setCommand(["/bin/sh", "-c",])
        .setArgs(["echo sleeping; date; sleep ${EXPECTED_BASELINE_DURATION_SECONDS+30}; echo sleep done; date;" +
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

    static final private DEFERRED_POST_LOCK_CLIENT_DEP = createAndRegisterDeployment()
        .setName(DEFERRED_POST_LOCK_DEP_NAME)
        .setImage(NGINX_IMAGE)
        .addLabel("app", DEFERRED_POST_LOCK_DEP_NAME)
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

    // returns whether true condition was achieved
    boolean retryUntilTrue(Closure<Boolean> closure, int retries) {
        Timer timer = new Timer(retries, 10)
        while (timer.IsValid()) {
            def result = closure()
            if (result) {
                return true
            }
        }
        return false
    }

    // validateBaseline checks that `expectedPeers` are present in the baseline and `explicitMissingPeers` are not.
    // Any other peer found is going to be ignored.
    //
    // Apparently there is a TCP connection via port 9537 that gets started in OpenShift clusters against any pod with
    // exposed ports. This was causing the test to fail since the expected baseline didn't match the size of the actual.
    // Although the anomalous flow filtering was working correctly, the additional flow shown in the baseline was coming
    // from this OpenShift connection in port 9537. To fix the issue, the split between `expectedPeers` and
    // `explicitMissingPeers` was introduced.
    // Check issues ROX-11142 and PR#2459 for more information.
    def validateBaseline(NetworkBaselineOuterClass.NetworkBaseline baseline,
                         long beforeCreate,
                         long justAfterCreate,
                         List<Tuple2<String, Boolean>> mustBeInBaseline,
                         List<String> mustNotBeInBaseline) {
        assert baseline.getObservationPeriodEnd().getSeconds() > beforeCreate - CLOCK_SKEW_ALLOWANCE_SECONDS
        assert baseline.getObservationPeriodEnd().getSeconds() <
            justAfterCreate + EXPECTED_BASELINE_DURATION_SECONDS + CLOCK_SKEW_ALLOWANCE_SECONDS
        assert baseline.getForbiddenPeersCount() == 0

        for (def i = 0; i < mustBeInBaseline.size(); i++) {
            def expectedPeerID = mustBeInBaseline.get(i).getFirst()
            def expectedPeerIngress = mustBeInBaseline.get(i).getSecond()
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

        for (def checkMissingId : mustNotBeInBaseline) {
            assert !baseline.getPeersList().any { it.getEntity().getInfo().getId() == checkMissingId }
        }
        return true
    }

    def cleanup() {
        log.info "Deleting the deployments."
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Tag("NetworkBaseline")
    @Timeout(1600)
    def "Verify network baseline functionality"() {
        when:
        "Create initial set of deployments, wait for baseline to populate"
        def beforeDeploymentCreate = System.currentTimeSeconds()
        batchCreate([SERVER_DEP, BASELINED_CLIENT_DEP])
        def justAfterDeploymentCreate = System.currentTimeSeconds()

        def serverDeploymentID = SERVER_DEP.deploymentUid
        assert serverDeploymentID != null

        def baselinedClientDeploymentID = BASELINED_CLIENT_DEP.deploymentUid
        assert baselinedClientDeploymentID != null

        Timestamp epoch = Timestamp.newBuilder().setSeconds(0).build()

        assert NetworkGraphUtil.checkForEdge(baselinedClientDeploymentID, serverDeploymentID, epoch, 180)

        // Now create the anomalous deployment
        batchCreate([ANOMALOUS_CLIENT_DEP])

        def anomalousClientDeploymentID = ANOMALOUS_CLIENT_DEP.deploymentUid
        assert anomalousClientDeploymentID != null
        log.info "Deployment IDs Server: ${serverDeploymentID}, " +
            "Baselined client: ${baselinedClientDeploymentID}, Anomalous client: ${anomalousClientDeploymentID}"

        assert NetworkGraphUtil.checkForEdge(anomalousClientDeploymentID, serverDeploymentID, epoch,
            EXPECTED_BASELINE_DURATION_SECONDS + 180, "Namespace:qa")

        def serverBaseline = evaluateWithRetry(30, 4) {
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
        log.info "Anomalous Baseline: ${anomalousClientBaseline}"
        def baselinedClientBaseline = NetworkBaselineService.getNetworkBaseline(baselinedClientDeploymentID)
        assert baselinedClientDeploymentID

        // Deployment IDs that must be explicitly check that are missing from server baseline
        def mustNotBeInBaseline = [anomalousClientDeploymentID]

        then:
        "Validate server baseline"
        // The anomalous client->server connection should not be baselined since the anonymous client
        // sleeps for a time period longer than the observation period before connecting to the server.
        validateBaseline(serverBaseline, beforeDeploymentCreate, justAfterDeploymentCreate,
            [new Tuple2<String, Boolean>(baselinedClientDeploymentID, true)], mustNotBeInBaseline)
        validateBaseline(anomalousClientBaseline, beforeDeploymentCreate, justAfterDeploymentCreate, [], [])
        validateBaseline(baselinedClientBaseline, beforeDeploymentCreate, justAfterDeploymentCreate,
            [new Tuple2<String, Boolean>(serverDeploymentID, false)], []
        )

        when:
        "Create another deployment, ensure it gets baselined"
        def beforeDeferredCreate = System.currentTimeSeconds()
        batchCreate([DEFERRED_BASELINED_CLIENT_DEP])
        def justAfterDeferredCreate = System.currentTimeSeconds()

        def deferredBaselinedClientDeploymentID = DEFERRED_BASELINED_CLIENT_DEP.deploymentUid
        assert deferredBaselinedClientDeploymentID != null
        log.info "Deferred Baseline: ${deferredBaselinedClientDeploymentID}"

        // Waiting on it to come out of observation.
        def deferredBaselinedClientBaseline = evaluateWithRetry(30, 4) {
            def baseline = NetworkBaselineService.getNetworkBaseline(deferredBaselinedClientDeploymentID)
            def now = System.currentTimeSeconds()
            if (baseline.getObservationPeriodEnd().getSeconds() > now) {
                throw new RuntimeException(
                    "Baseline ${deferredBaselinedClientDeploymentID} is in observation. Baseline is ${baseline}"
                )
            }
            return baseline
        }
        assert deferredBaselinedClientBaseline

        assert NetworkGraphUtil.checkForEdge(deferredBaselinedClientDeploymentID, serverDeploymentID, null, 180)
        // Make sure peers have been added to the serverBaseline
        serverBaseline = evaluateWithRetry(30, 4) {
            def baseline = NetworkBaselineService.getNetworkBaseline(serverDeploymentID)
            if (baseline.getPeersCount() < 2) {
                throw new RuntimeException(
                    "Not enough peers in baseline for deployment ${serverDeploymentID} yet. Baseline is ${baseline}"
                )
            }
            return baseline
        }
        assert serverBaseline

        then:
        "Validate the updated baselines"
        validateBaseline(serverBaseline, beforeDeploymentCreate, justAfterDeploymentCreate,
            [new Tuple2<String, Boolean>(baselinedClientDeploymentID, true),
             // Currently, we add cons to the baseline if it's within the observation period
             // of _at least_ one of the deployments. Therefore, the deferred client->server connection
             // gets added since it's within the deferred client's observation period, and
             // the server's baseline is modified as well since we keep things consistent.
             new Tuple2<String, Boolean>(deferredBaselinedClientDeploymentID, true),
            ], mustNotBeInBaseline
        )
        validateBaseline(deferredBaselinedClientBaseline, beforeDeferredCreate, justAfterDeferredCreate,
            [new Tuple2<String, Boolean>(serverDeploymentID, false)], [])

        when:
        "Create another deployment, ensure it DOES NOT get added to serverDeploymentID due to user lock"
        NetworkBaselineService.lockNetworkBaseline(serverDeploymentID)

        batchCreate([DEFERRED_POST_LOCK_CLIENT_DEP])

        def postLockClientDeploymentID = DEFERRED_POST_LOCK_CLIENT_DEP.deploymentUid
        assert postLockClientDeploymentID != null
        log.info "Post Lock Deployment: ${postLockClientDeploymentID}"

        // Waiting on it to come out of observation.
        def postLockClientBaseline = evaluateWithRetry(30, 4) {
            def baseline = NetworkBaselineService.getNetworkBaseline(postLockClientDeploymentID)
            def now = System.currentTimeSeconds()
            if (baseline.getObservationPeriodEnd().getSeconds() > now) {
                throw new RuntimeException(
                    "Baseline ${postLockClientDeploymentID} is not out of observation yet. Baseline is ${baseline}"
                )
            }
            return baseline
        }
        assert postLockClientBaseline

        assert NetworkGraphUtil.checkForEdge(postLockClientDeploymentID, serverDeploymentID, null, 180)

        // Grab the latest server baseline for validation
        serverBaseline = NetworkBaselineService.getNetworkBaseline(serverDeploymentID)
        assert serverBaseline

        then:
        "Validate the locked baselines"
        // Post lock should not be added as a peer because serverBaseline is locked.
        validateBaseline(serverBaseline, beforeDeploymentCreate, justAfterDeploymentCreate,
            [new Tuple2<String, Boolean>(baselinedClientDeploymentID, true),
             new Tuple2<String, Boolean>(deferredBaselinedClientDeploymentID, true),
            ], mustNotBeInBaseline
        )
        validateBaseline(postLockClientBaseline, beforeDeferredCreate, justAfterDeferredCreate,
            [], [])

        when:
        "Verify user get for non-existent baseline"
        def beforeUserServiceDeploymentCreate = System.currentTimeSeconds()
        batchCreate([USER_DEP])
        def justAfterUserServiceDeploymentCreate = System.currentTimeSeconds()

        def userReqBaselineServerDeploymentID = USER_DEP.deploymentUid
        assert userReqBaselineServerDeploymentID != null

        log.info "Deployment IDs Server: ${userReqBaselineServerDeploymentID}"

        // Get the server baseline to simulate a user asking for a baseline prior to
        // observation ending.  This will generate a baseline at the time of request
        // instead of after observation.
        def userReqServerBaseline = NetworkBaselineService.getNetworkBaseline(userReqBaselineServerDeploymentID)
        // Ensure the baseline is STILL in the observation window
        def now = System.currentTimeSeconds()
        assert userReqServerBaseline.getObservationPeriodEnd().getSeconds() > now

        // Add a client deployment
        def beforeClientDeploymentCreate = System.currentTimeSeconds()
        batchCreate([BASELINED_USER_CLIENT_DEP])
        def justAfterClientDeploymentCreate = System.currentTimeSeconds()

        def userRequestedBaselinedClientDeploymentID = BASELINED_USER_CLIENT_DEP.deploymentUid
        assert userRequestedBaselinedClientDeploymentID != null
        log.info "Client deployment: ${userRequestedBaselinedClientDeploymentID}"

        assert retryUntilTrue({
            return NetworkGraphUtil.checkForEdge(
                userRequestedBaselinedClientDeploymentID,
                userReqBaselineServerDeploymentID)
                    .any { it.targetID == userReqBaselineServerDeploymentID }
        }, 15)

        // Grab the network baseline for the client.
        def userReqBaselinedClientBaseline =
            NetworkBaselineService.getNetworkBaseline(userRequestedBaselinedClientDeploymentID)
        assert userReqBaselinedClientBaseline

        // Grab a fresh copy of the userReqServerBaseline after the client connection has been added.
        userReqServerBaseline = NetworkBaselineService.getNetworkBaseline(userReqBaselineServerDeploymentID)

        log.info "Server Baseline: ${userReqServerBaseline}"
        log.info "Client Baseline: ${userReqBaselinedClientBaseline}"

        then:
        "Validate user requested server baseline"
        // The client->server connection should be baselined since the client as the
        // connection occurred during the observation window.
        validateBaseline(userReqServerBaseline, beforeUserServiceDeploymentCreate, justAfterUserServiceDeploymentCreate,
            [new Tuple2<String, Boolean>(userRequestedBaselinedClientDeploymentID, true)], [])
        validateBaseline(userReqBaselinedClientBaseline, beforeClientDeploymentCreate, justAfterClientDeploymentCreate,
            [new Tuple2<String, Boolean>(userReqBaselineServerDeploymentID, false)], []
        )
    }
}
