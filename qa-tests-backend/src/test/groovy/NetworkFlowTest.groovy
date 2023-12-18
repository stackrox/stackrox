import static io.restassured.RestAssured.given
import static util.Helpers.withRetry

import io.grpc.StatusRuntimeException
import io.restassured.response.Response
import orchestratormanager.OrchestratorTypes
import org.yaml.snakeyaml.Yaml

import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.NetworkGraph
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.NetworkNode
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GenerateNetworkPoliciesRequest.DeleteExistingPoliciesMode
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.NetworkFlowOuterClass.L4Protocol
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntityInfo.Type
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicyModification

import common.Constants
import objects.DaemonSet
import objects.Deployment
import objects.Edge
import objects.K8sServiceAccount
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Service
import services.ClusterService
import services.DeploymentService
import services.NetworkGraphService
import services.NetworkPolicyService
import util.Env
import util.Helpers
import util.NetworkGraphUtil
import util.Timer

import org.junit.Assume
import spock.lang.Ignore
import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Stepwise
import spock.lang.Tag
import spock.lang.Unroll

// TODO(ROX-13739): Re-enable these tests in compatibility-test step
@Stepwise
@Tag("PZ")
class NetworkFlowTest extends BaseSpecification {

    // Deployment names
    static final private String UDPCONNECTIONTARGET = "udp-connection-target"
    static final private String TCPCONNECTIONTARGET = "tcp-connection-target"
    static final private String NGINXCONNECTIONTARGET = "nginx-connection-target"
    static final private String UDPCONNECTIONSOURCE = "udp-connection-source"
    static final private String TCPCONNECTIONSOURCE = "tcp-connection-source"
    //static final private String ICMPCONNECTIONSOURCE = "icmp-connection-source"
    static final private String NOCONNECTIONSOURCE = "no-connection-source"
    static final private String SHORTCONSISTENTSOURCE = "short-consistent-source"
    static final private String SINGLECONNECTIONSOURCE = "single-connection-source"
    static final private String MULTIPLEPORTSCONNECTION = "two-ports-connect-source"
    static final private String EXTERNALDESTINATION = "external-destination-source"

    // Other namespace
    static final private String OTHER_NAMESPACE = "qa2"

    static final private String SOCAT_DEBUG = "-d -d -v"

    // Target deployments
    @Shared
    private List<Deployment> targetDeployments

    def buildTargetDeployments() {
        return [
            new Deployment()
                    .setName(UDPCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addPort(8080, "UDP")
                    .addLabel("app", UDPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["socat "+SOCAT_DEBUG+" UDP-RECV:8080 STDOUT",]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addPort(80)
                    .addPort(8080)
                    .addLabel("app", TCPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat "+SOCAT_DEBUG+" TCP-LISTEN:80,fork STDOUT & " +
                                      "socat "+SOCAT_DEBUG+" TCP-LISTEN:8080,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(NGINXCONNECTIONTARGET)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addPort(80)
                    .addLabel("app", NGINXCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCreateLoadBalancer(!
                        (Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x"))
                    .setCreateRoute(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT),
        ]
    }

    // Source deployments
    @Shared
    private List<Deployment> sourceDeployments

    def buildSourceDeployments() {
        return [
            new Deployment()
                    .setName(NOCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addLabel("app", NOCONNECTIONSOURCE),
            new Deployment()
                    .setName(SHORTCONSISTENTSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                    .addLabel("app", SHORTCONSISTENTSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
                                      "do wget -S -T 2 http://${NGINXCONNECTIONTARGET}; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(SINGLECONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                    .addLabel("app", SINGLECONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["wget -S -T 2 http://${NGINXCONNECTIONTARGET} && " +
                                      "while sleep 30; do echo hello; done" as String,]),
            new Deployment()
                    .setName(UDPCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addLabel("app", UDPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${UDPCONNECTIONSOURCE}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN UDP:${UDPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONSOURCE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addLabel("app", TCPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(MULTIPLEPORTSCONNECTION)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addLabel("app", MULTIPLEPORTSCONNECTION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN TCP:${TCPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(EXTERNALDESTINATION)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                    .addLabel("app", EXTERNALDESTINATION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS}; " +
                                      "do wget -S -T 2 http://www.google.com; " +
                                      "done" as String,]),
            new Deployment()
                    .setName("${TCPCONNECTIONSOURCE}-qa2")
                    .setNamespace(OTHER_NAMESPACE)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:socat")
                    .addLabel("app", "${TCPCONNECTIONSOURCE}-qa2")
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}-qa2\" | " +
                                      "socat "+SOCAT_DEBUG+" -s STDIN "+
                                         "TCP:${TCPCONNECTIONTARGET}.qa.svc.cluster.local:80; " +
                                      "done" as String,]),
        ]
    }

    @Shared
    private List<Deployment> deployments

    def createDeployments() {
        targetDeployments = buildTargetDeployments()
        orchestrator.batchCreateDeployments(targetDeployments)
        for (Deployment d : targetDeployments) {
            assert Services.waitForDeployment(d)
        }
        sourceDeployments = buildSourceDeployments()
        orchestrator.batchCreateDeployments(sourceDeployments)
        for (Deployment d : sourceDeployments) {
            assert Services.waitForDeployment(d)
        }
        deployments = sourceDeployments + targetDeployments
        //
        // Commenting out ICMP test setup for now
        // See ROX-635
        //
        /*
        def nginxIp = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.pods?.get(0)?.podIP
        Deployment icmp = new Deployment()
                .setName(ICMPCONNECTIONSOURCE)
                .setImage("ubuntu")
                .addLabel("app", ICMPCONNECTIONSOURCE)
                .setCommand(["/bin/sh", "-c",])
                .setArgs(["apt-get update && " +
                                  "apt-get install iputils-ping -y && " +
                                  "ping ${nginxIp}" as String,])
        orchestrator.createDeployment(icmp)
        DEPLOYMENTS.add(icmp)
        */
    }

    def setupSpec() {
        orchestrator.createNamespace(OTHER_NAMESPACE)
        orchestrator.createImagePullSecret(
                "quay",
                Env.mustGet("REGISTRY_USERNAME"),
                Env.mustGet("REGISTRY_PASSWORD"),
                OTHER_NAMESPACE,
                "https://quay.io"
        )
        orchestrator.createServiceAccount(
                new K8sServiceAccount(
                        name: "default",
                        namespace: OTHER_NAMESPACE,
                        imagePullSecrets: ["quay"]
                )
        )

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
        orchestrator.deleteNamespace(OTHER_NAMESPACE)
        orchestrator.waitForNamespaceDeletion(OTHER_NAMESPACE)
    }

    def cleanupSpec() {
        destroyDeployments()
    }

    @Tag("NetworkFlowVisualization")
    // TODO: additional handling may be needed for P/Z, skipping for 1st release
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify one-time connections show at first and are closed after the afterglow period"() {
        given:
        "Two deployments, A and B, where B communicates to A a single time during initial deployment"
        String targetUid = deployments.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = deployments.find { it.name == SINGLECONNECTIONSOURCE }?.deploymentUid
        assert sourceUid != null

        when:
        "Check for edge in network graph"
        log.info "Checking for edge between ${SINGLECONNECTIONSOURCE} and ${NGINXCONNECTIONTARGET}"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "Wait for collector update and fetch graph again to confirm connection dropped"
        // 65 seconds is the grace period for updates because a closed connection is subject to
        // afterglow and the rate at which collector and sensor sends network flows (30s respectively).
        // The afterglow period in testing is 15s so the max time for the close message to propagate is
        // 30s in collector, 30s in sensor, plus 5s of buffer time for transit/storage.
        // The network graph continually returns timestamp.Now() if the lastSeenTime is nil.
        assert waitForEdgeToBeClosed(edges.get(0), 65)
    }

    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("NetworkFlowVisualization")
    // TODO: additional handling may be needed for P/Z, skipping for 1st release
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify connections between StackRox Services"() {
        when:
        "Fetch uIDs for the central, sensor, and collector services, if present"
        String centralUid = orchestrator.getDeploymentId(new Deployment(name: "central", namespace: "stackrox"))
        assert centralUid != null
        String sensorUid = orchestrator.getDeploymentId(new Deployment(name: "sensor", namespace: "stackrox"))
        assert sensorUid != null
        String collectorUid = orchestrator.getDaemonSetId(new DaemonSet(name: "collector", namespace: "stackrox"))
        // collector id *can* be null, so no assert

        then:
        "Check for edge between sensor and central"
        log.info "Checking for edge between sensor and central"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(sensorUid, centralUid)
        assert edges

        then:
        "Check for edge between collector and sensor, if collector is installed"
        if (collectorUid != null) {
            log.info "Checking for edge between collector and sensor"
            edges = NetworkGraphUtil.checkForEdge(collectorUid, sensorUid)
            assert edges
        }
    }

    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("NetworkFlowVisualization")
    // TODO: additional handling may be needed for P/Z, skipping for 1st release
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    @IgnoreIf({ true }) // ROX-16849 this test is flaking in many cluster flavors
    def "Verify ports are greater than 0"() {
        given:
        "ACS is running"
        def graph = NetworkGraphService.getNetworkGraph(null, null)
        def nodes = graph.nodesList

        nodes.each { node ->
            node.getOutEdgesMap().each { key, value ->
                value.propertiesList.each { property ->
                    assert property.port > 0
                }
            }
        }
    }

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("NetworkFlowVisualization")
    def "Verify connections can be detected: #protocol"() {
        given:
        "Two deployments, A and B, where B communicates to A via #protocol"
        String targetUid = deployments.find { it.name == targetDeployment }?.deploymentUid
        assert targetUid != null
        String sourceUid = deployments.find { it.name == sourceDeployment }?.deploymentUid
        assert sourceUid != null

        expect:
        "Check for edge in network graph"
        log.info "Checking for edge between ${sourceDeployment} and ${targetDeployment}"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(sourceUid, targetUid)

        assert edges
        assert edges.get(0).protocol == protocol
        assert deployments.find { it.name == targetDeployment }?.ports?.keySet()?.contains(edges.get(0).port)

        where:
        "Data is:"

        sourceDeployment     | targetDeployment      | protocol
        UDPCONNECTIONSOURCE  | UDPCONNECTIONTARGET   | L4Protocol.L4_PROTOCOL_UDP
        TCPCONNECTIONSOURCE  | TCPCONNECTIONTARGET   | L4Protocol.L4_PROTOCOL_TCP
        //ICMPCONNECTIONSOURCE | NGINXCONNECTIONTARGET | L4Protocol.L4_PROTOCOL_ICMP
    }

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("NetworkFlowVisualization")
    def "Verify listen port availability matches feature flag: #targetDeployment"() {
        given:
        "Deployment with listening port"
        String targetUid = deployments.find { it.name == targetDeployment }?.deploymentUid
        assert targetUid

        expect:
        "Check for (absence of) listening port info"
        def node = getNode(targetUid, expectedListenPorts.size() > 0)
        assert node
        assert (node.listenPorts(L4Protocol.L4_PROTOCOL_TCP)*.port as Set) == (expectedListenPorts as Set)

        where:
        "Data is:"

        targetDeployment      | expectedListenPorts
        TCPCONNECTIONTARGET   | [80, 8080]
        NGINXCONNECTIONTARGET | [80]
        NOCONNECTIONSOURCE    | [80]
        TCPCONNECTIONSOURCE   | []
    }

    @Tag("NetworkFlowVisualization")
    // TODO: additional handling may be needed for P/Z, skipping for 1st release
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify connections with short consistent intervals between 2 deployments"() {
        given:
        "Two deployments, A and B, where B communicates to A in short consistent intervals"
        String targetUid = deployments.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = deployments.find { it.name == SHORTCONSISTENTSOURCE }?.deploymentUid
        assert sourceUid != null

        when:
        "Check for edge in network graph"
        log.info "Checking for edge between ${SHORTCONSISTENTSOURCE} and ${NGINXCONNECTIONTARGET}"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "Wait for collector update and fetch graph again to confirm short interval connections remain"
        assert waitForEdgeUpdate(edges.get(0), 90)
    }

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("NetworkFlowVisualization")
    def "Verify network graph when filtered on \"#filter\" and scoped to \"#scope\" #desc"() {
        given:
        "Orchestrator components exists"
        def allDeps = NetworkGraphUtil.getDeploymentsAsGraphNodes()

        when:
        "Network graph is filtered on \"#filter\" and scoped to \"#scope\""
        def graph = NetworkGraphService.getNetworkGraph(null, filter, scope)

        then:
        "Network graph #desc"
        assert NetworkGraphUtil.verifyGraphFilterAndScope(graph, allDeps.nonOrchestratorDeployments,
                allDeps.orchestratorDeployments, nonOrchestratorDepsShouldExist, orchestratorDepsShouldExist)

        when:
        "Network policy graph is filtered on \"#filter\" and scoped to \"#scope\""
        graph = NetworkPolicyService.getNetworkPolicyGraph(filter, scope)

        then:
        "Network policy graph #desc"
        assert NetworkGraphUtil.verifyGraphFilterAndScope(graph, allDeps.nonOrchestratorDeployments,
                allDeps.orchestratorDeployments, nonOrchestratorDepsShouldExist, orchestratorDepsShouldExist)

        where:
        "Data is:"

        filter                         | scope                          | orchestratorDepsShouldExist |
                nonOrchestratorDepsShouldExist | desc
        ""                             | "Orchestrator Component:false" | false                       |
                true | "contains non-orchestrator deployments only"
        "Orchestrator Component:false" | ""                             | true                        |
                true | "contains non-orchestrator deployments and connected orchestrator deployments"
        "Orchestrator Component:true"  | "Orchestrator Component:false" | false                       |
                false | "contains no deployments"
        "Orchestrator Component:false" | "Orchestrator Component:true"  | false                       |
                false | "contains no deployments"
        "Namespace:stackrox"           | "Orchestrator Component:false" | false                       |
                true | "contains stackrox deployments only"
    }

    @Tag("BAT")
    @Tag("NetworkFlowVisualization")
    def "Verify network flows with graph filtering"() {
        given:
        "Two deployments, A and B, where B communicates to A"
        String sourceUid = deployments.find { it.name == TCPCONNECTIONSOURCE }?.deploymentUid
        assert sourceUid != null
        String targetUid = deployments.find { it.name == TCPCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null

        when:
        "Check for edge in network graph"
        log.info "Checking for edge between ${TCPCONNECTIONSOURCE} and ${TCPCONNECTIONTARGET}"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "Wait for collector update and fetch graph again to confirm short interval connections remain"
        assert waitForEdgeUpdate(edges.get(0), 90)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify connections to external sources"() {
        given:
        "Deployment A, where A communicates to an external target"
        String deploymentUid = deployments.find { it.name == EXTERNALDESTINATION }?.deploymentUid
        assert deploymentUid != null

        expect:
        "Check for edge in network graph"
        log.info "Checking for edge from ${EXTERNALDESTINATION} to external target"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(deploymentUid, Constants.INTERNET_EXTERNAL_SOURCE_ID)
        assert edges
    }

    @Tag("NetworkFlowVisualization")
    @Ignore("ROX-19615")
    def "Verify connections from external sources"() {
        given:
        "Deployment A, where an external source communicates to A"
        String deploymentUid = deployments.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert deploymentUid != null
        String targetUrl
        if (Env.mustGetOrchestratorType() == OrchestratorTypes.K8S) {
            String deploymentIP = deployments.find { it.name == NGINXCONNECTIONTARGET }?.loadBalancerIP
            assert deploymentIP != null
            targetUrl = "http://${deploymentIP}"
        } else if (Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT) {
            String routeHost = deployments.find { it.name == NGINXCONNECTIONTARGET }?.routeHost
            assert routeHost != null
            targetUrl = "http://${routeHost}"
        } else {
            throw new RuntimeException("Unexpected OrchestratorType")
        }

        when:
        log.info "Generate initial traffic to the target deployment ${NGINXCONNECTIONTARGET}"
        def initialResponse = doHTTPGetExpectCode(targetUrl, 200)
        assert initialResponse?.getStatusCode() == 200

        then:
        "Check for edge in network graph"
        withRetry(5, 20) {
            log.info "Retry traffic to the target deployment ${NGINXCONNECTIONTARGET}"
            def response = doHTTPGetExpectCode(targetUrl, 200)
            assert response?.getStatusCode() == 200

            log.info "Checking for edge from external to ${NGINXCONNECTIONTARGET}"
            // Only on OpenShift 4.12, the edge will not show from EXTERNAL_SOURCE, but instead from
            // router-default deployment in openshift-ingress namespace.
            if (ClusterService.isOpenShift4()) {
                log.info("Searching for edge coming from OpenShift ingress router to ${deploymentUid}")
                SearchServiceOuterClass.RawQuery query = SearchServiceOuterClass.RawQuery.newBuilder()
                        .setQuery("Namespace:openshift-ingress")
                        .build()
                def ingressDeployments = DeploymentService.listDeploymentsSearch(query).deploymentsList
                def defaultRouterId = ingressDeployments.find { it.getName() == "router-default" }.id
                def rEdges = NetworkGraphUtil.checkForEdge(defaultRouterId, deploymentUid, null, 180)
                if (rEdges != null) {
                    log.info("Found edge coming from OpenShift ingress router")
                    return
                }
                log.warn("Edge coming from OpenShift ingress router to ${deploymentUid} not found")
                // Debug dump of all router edges
                def currentGraph = NetworkGraphService.getNetworkGraph()
                def index = currentGraph.nodesList.findIndexOf { node -> node.deploymentName == defaultRouterId }
                List<NetworkNode> outNodesRouter = currentGraph.nodesList.findAll { node ->
                    node.outEdgesMap.containsKey(index)
                }
                log.debug("All edges of 'router-default' ${defaultRouterId}: ${outNodesRouter}")
            }
            log.info("Searching for edge coming from INTERNET_EXTERNAL_SOURCE_ID " +
                "(${Constants.INTERNET_EXTERNAL_SOURCE_ID}) to ${deploymentUid}")
            List<Edge> edges =
                    NetworkGraphUtil.checkForEdge(Constants.INTERNET_EXTERNAL_SOURCE_ID, deploymentUid, null, 180)
            if (edges == null || edges.size() == 0) {
                // Debug dump of all INTERNET_EXTERNAL_SOURCE_ID edges
                def currentGraph = NetworkGraphService.getNetworkGraph()
                def index = currentGraph.nodesList.findIndexOf {
                    node -> node.deploymentName == Constants.INTERNET_EXTERNAL_SOURCE_ID
                }
                List<NetworkNode> outNodes = currentGraph.nodesList.findAll { node ->
                    node.outEdgesMap.containsKey(index)
                }
                log.debug("All edges of 'INTERNET_EXTERNAL_SOURCE_ID' " +
                    "${Constants.INTERNET_EXTERNAL_SOURCE_ID}: ${outNodes}")
            }
            assert edges
        }
    }

    def doHTTPGetExpectCode(String targetUrl, int code) {
        Response response = null
        Timer t = new Timer(12, 5)
        while (response?.statusCode() != code && t.IsValid()) {
            try {
                log.info "Trying HTTP Get to ${targetUrl}..."
                response = given().get(targetUrl)
            } catch (Exception e) {
                log.warn("Failed calling ${targetUrl}. Trying again in 5 sec...", e)
            }
        }
        log.info "Response: " + response.asString()
        return response
    }

    @Tag("NetworkFlowVisualization")
    def "Verify intra-cluster connection via external IP"() {
        // We changed the test to reflect the NetworkGraph's current behavior. Communication between two deployments
        // through a LoadBalancer shows an edge from 'External Entities', not an edge between the two deployments.
        // ROX-17936 should address whether we revert to the old behavior or we maintain this new behavior.
        // We do not test this on OCP.
        // Manual testing on OCP shows the same behavior but on prow times out most of the time.
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)
        given:
        "Deployment A, exposed via LB"
        String deploymentUid = deployments.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert deploymentUid != null
        String deploymentIP = deployments.find { it.name == NGINXCONNECTIONTARGET }?.loadBalancerIP
        assert deploymentIP != null

        when:
        "create a new deployment that talks to A via the LB IP"
        def newDeployment = new Deployment()
                .setName("talk-to-lb-ip")
                .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                .addLabel("app", "talk-to-lb-ip")
                .setCommand(["/bin/sh", "-c",])
                .setArgs(["while sleep 5; do wget -S -T 2 http://"+deploymentIP+"; done"])

        orchestrator.createDeployment(newDeployment)
        assert Services.waitForDeployment(newDeployment)
        assert newDeployment.deploymentUid

        then:
        "Check for edge in network graph"
        withRetry(2, 10) {
            log.info "Checking for edge from internal to ${NGINXCONNECTIONTARGET} using its external address"
            List<Edge> edgesToExternalSource = NetworkGraphUtil.checkForEdge(
                    Constants.INTERNET_EXTERNAL_SOURCE_ID,
                    deploymentUid, null, 180)
            assert edgesToExternalSource
        }

        cleanup:
        "remove the new deployment"
        if (newDeployment) {
            orchestrator.deleteDeployment(newDeployment)
        }
    }

    @Tag("NetworkFlowVisualization")
    def "Verify no connections between 2 deployments"() {
        given:
        "Two deployments, A and B, where neither communicates to the other"
        String targetUid = deployments.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = deployments.find { it.name == NOCONNECTIONSOURCE }?.deploymentUid
        assert sourceUid != null

        expect:
        "Assert connection states"
        log.info "Checking for NO edge between ${NOCONNECTIONSOURCE} and ${NGINXCONNECTIONTARGET}"
        assert !NetworkGraphUtil.checkForEdge(sourceUid, targetUid, null, 30)
    }

    @Tag("NetworkFlowVisualization")
    def "Verify connections between two deployments on 2 separate ports shows both edges in the graph"() {
        given:
        "Two deployments, A and B, where B communicates to A on 2 different ports"
        String targetUid = deployments.find { it.name == TCPCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = deployments.find { it.name == MULTIPLEPORTSCONNECTION }?.deploymentUid
        assert sourceUid != null

        when:
        "Check for edge in entwork graph"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "Assert that there are 2 connection edges"
        assert edges.size() == 2
    }

    @Tag("NetworkFlowVisualization")
    // TODO: additional handling may be needed for P/Z, skipping for 1st release
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify cluster updates can block flow connections from showing"() {
        // ROX-7153 - EKS cannot NetworkPolicy (RS-178)
        Assume.assumeFalse(ClusterService.isEKS())
        // ROX-7153 - AKS cannot tolerate NetworkPolicy (RS-179)
        Assume.assumeFalse(ClusterService.isAKS())

        given:
        "Two deployments, A and B, where B communicates to A"
        String targetUid = deployments.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = deployments.find { it.name == SHORTCONSISTENTSOURCE }?.deploymentUid
        assert sourceUid != null

        and:
        "The edge is found before blocked"
        log.info "Checking for edge between ${SHORTCONSISTENTSOURCE} and ${NGINXCONNECTIONTARGET}"
        List<Edge> edges = NetworkGraphUtil.checkForEdge(sourceUid, targetUid)
        assert edges

        when:
        "apply network policy to block ingress to A"
        NetworkPolicy policy = new NetworkPolicy("deny-all-traffic-to-a")
                .setNamespace("qa")
                .addPodSelector(["app":NGINXCONNECTIONTARGET])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy)
        log.info "Sleeping 60s to allow policy to propagate and flows to update after propagation"
        sleep 60000

        and:
        "Get the latest edge"
        log.info "Checking for latest edge between ${SHORTCONSISTENTSOURCE} and ${NGINXCONNECTIONTARGET}"
        edges = NetworkGraphUtil.checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "make sure edge does not get updated"
        //Use one cadence buffer to account for additional edges coming in through the data pipeline
        assert !waitForEdgeUpdate(edges.get(0), 60, NetworkGraphUtil.NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS)

        cleanup:
        "remove policy"
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy)
        }
    }

    @Tag("NetworkFlowVisualization")
    def "Verify edge timestamps are never in the future, or before start of flow tests"() {
        given:
        "Get current state of edges and current timestamp"
        def queryString = "Deployment:" + deployments.name.join(",")
        NetworkGraph currentGraph = NetworkGraphService.getNetworkGraph(null, queryString)
        long currentTime = System.currentTimeMillis()

        expect:
        "Check timestamp for each edge"
        for (Edge edge : NetworkGraphUtil.findEdges(currentGraph, null, null)) {
            assert edge.lastActiveTimestamp <= currentTime + 2000 //allow up to 2 sec leeway
            assert edge.lastActiveTimestamp >= testSpecStartTimeMillis
        }
    }

    @Tag("BAT")
    def "Verify generated network policies"() {
        // TODO(RS-178): EKS cannot NetworkPolicy
        Assume.assumeFalse(ClusterService.isEKS())

        given:
        "Get current state of deployed namespaces"
        List<String> deployedNamespaces = deployments*.namespace

        and:
        "delete a deployment"
        Deployment delete = deployments.find { it.name == NOCONNECTIONSOURCE }
        orchestrator.deleteDeployment(delete)
        Services.waitForSRDeletion(delete)

        when:
        "Network graph contains all expected targets"
        NetworkGraph currentGraph = null

        // This is a map of "deploymentName" -> expectedNumberOfEdges.
        // It allows us to know when the network graph has reached a desired state and the proper assertions can start.
        def targets = [(TCPCONNECTIONTARGET):3, (UDPCONNECTIONTARGET):1]
        targets.each { deploymentName, expectedEdges ->
            // Make sure that we have enough time to populate the graph: 30s for sensor + some extra.
            // On the first run, this test benefits from the state left over by previous tests.
            // However, if the first run fails, the env is recreated and the test is run immediately afterwards.
            // The system requires time for the changes to propagate to sensor, thus we have an additional retry here.
            log.info "Verifying graph edges of ${deploymentName}"
            Helpers.withRetry(6, 20) { retry ->
                // The graph needs to be re-retrieved on each retry
                currentGraph = NetworkGraphService.getNetworkGraph()

                def index = currentGraph.nodesList.findIndexOf { node -> node.deploymentName == deploymentName }
                List<NetworkNode> outNodes = currentGraph.nodesList.findAll { node ->
                    node.outEdgesMap.containsKey(index)
                }
                def allowAllIngress = deployments.find { it.name == deploymentName }?.createLoadBalancer ||
                    currentGraph.nodesList.find { it.entity.type == Type.INTERNET }.outEdgesMap.containsKey(index)
                if (allowAllIngress) {
                    log.info "${deploymentName} has LB/External incoming traffic - ensure All Ingress allowed"
                } else {
                    assert outNodes.size() > 0 == expectedEdges > 0
                    log.info "${deploymentName} has incoming connections"
                    def sourceDeploymentsFromGraph = outNodes.findAll { it.deploymentName }*.deploymentName

                    log.debug("sourceDeploymentsFromGraph: {}", sourceDeploymentsFromGraph)
                    assert sourceDeploymentsFromGraph.size() == expectedEdges
                }
            }
        }

        and:
        "Generate Network Policies"
        NetworkPolicyModification modification = NetworkPolicyService.generateNetworkPolicies()
        Yaml parser = new Yaml()
        List yamls = []
        for (String yaml : modification.applyYaml.split("---")) {
            yamls.add(parser.load(yaml))
        }

        then:
        "verify generated netpols against the current graph state"
        yamls.each {
            assert it."metadata"."namespace" != "kube-system" &&
                    it."metadata"."namespace" != "kube-public"
        }
        yamls.findAll {
            deployedNamespaces.contains(it."metadata"."namespace")
        }.each { yaml ->
            String deploymentName =
                yaml."metadata"."name"["stackrox-generated-".length()..yaml."metadata"."name".length() - 1]
            assert deploymentName != NOCONNECTIONSOURCE
            assert yaml."metadata"."labels"."network-policy-generator.stackrox.io/generated"
            assert yaml."metadata"."namespace"
            def index = currentGraph.nodesList.findIndexOf { node -> node.deploymentName == deploymentName }
            def allowAllIngress = deployments.find { it.name == deploymentName }?.createLoadBalancer ||
                currentGraph.nodesList.find { it.entity.type == Type.INTERNET }.outEdgesMap.containsKey(index)
            List<NetworkNode> outNodes = currentGraph.nodesList.findAll { node ->
                node.outEdgesMap.containsKey(index)
            }
            def ingressPodSelectors = yaml."spec"."ingress".find { it.containsKey("from") } ?
                yaml."spec"."ingress".get(0)."from".findAll { it.containsKey("podSelector") } :
                null
            def ingressNamespaceSelectors = yaml."spec"."ingress".find { it.containsKey("from") } ?
                yaml."spec"."ingress".get(0)."from".findAll { it.containsKey("namespaceSelector") } :
                null
            if (allowAllIngress) {
                log.info "${deploymentName} has LB/External incoming traffic - ensure All Ingress allowed"
                assert yaml."spec"."ingress" == [[:]]
            } else if (outNodes.size() > 0) {
                log.info "${deploymentName} has incoming connections - " +
                    "ensure podSelectors/namespaceSelectors match sources from graph"
                def sourceDeploymentsFromGraph = outNodes.findAll { it.deploymentName }*.deploymentName
                def sourceDeploymentsFromNetworkPolicy = ingressPodSelectors.collect {
                    it."podSelector"."matchLabels"."app"
                }
                def sourceNamespacesFromNetworkPolicy = ingressNamespaceSelectors.collect {
                    it."namespaceSelector"."matchLabels"."namespace.metadata.stackrox.io/name"
                }.findAll { it != null }
                sourceNamespacesFromNetworkPolicy.addAll(ingressNamespaceSelectors.collect {
                    it."namespaceSelector"."matchLabels"."kubernetes.io/metadata.name"
                }).findAll { it != null }

                log.debug("sourceDeploymentsFromNetworkPolicy: {}", sourceDeploymentsFromNetworkPolicy)
                log.debug("sourceDeploymentsFromGraph: {}", sourceDeploymentsFromGraph)
                assert sourceDeploymentsFromNetworkPolicy.sort() == sourceDeploymentsFromGraph.sort()

                if (!deployedNamespaces.containsAll(sourceNamespacesFromNetworkPolicy)) {
                    log.info "Deployed namespaces do not contain all namespaces found in the network policy"
                    log.info "The network policy:"
                    log.info modification.toString()
                }
                assert deployedNamespaces.containsAll(sourceNamespacesFromNetworkPolicy)
            } else {
                log.info "${deploymentName} has no incoming connections - ensure ingress spec is empty"
                assert yaml."spec"."ingress" == [] || yaml."spec"."ingress" == null
            }
        }
    }

    // ROX-13903 - applyGeneratedNetworkPolicy() can hang when run via the
    // standalone test runner.
    @IgnoreIf({ !Env.IN_CI })
    @Unroll
    @Tag("BAT")
    def "Verify network policy generator apply/undo with delete modes: #deleteMode #note"() {
        given:
        "apply network policies to the system"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic-to-app")
                .setNamespace("qa")
                .addPodSelector(["app":NGINXCONNECTIONTARGET])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        NetworkPolicy policy2 = new NetworkPolicy("generated-deny-all-traffic-to-app")
                .setNamespace("qa")
                .addLabel("network-policy-generator.stackrox.io/generated", "true")
                .addPodSelector(["app":NGINXCONNECTIONTARGET])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def policyId1 = orchestrator.applyNetworkPolicy(policy1)
        def policyId2 = orchestrator.applyNetworkPolicy(policy2)
        assert NetworkPolicyService.waitForNetworkPolicy(policyId1)
        assert NetworkPolicyService.waitForNetworkPolicy(policyId2)

        and:
        "Get existing network policies from orchestrator"
        def preExistingNetworkPolicies = getQANetworkPoliciesNamesByNamespace(true)
        log.info "${preExistingNetworkPolicies}"

        expect:
        "actual policies should exist in generated response depending on delete mode"
        def modification = NetworkPolicyService.generateNetworkPolicies(deleteMode, 'Namespace:r/^qa2?$')
        assert !(NetworkPolicyService.applyGeneratedNetworkPolicy(modification) instanceof StatusRuntimeException)
        def appliedNetworkPolicies = getQANetworkPoliciesNamesByNamespace(true)
        log.info "${appliedNetworkPolicies}"

        Yaml parser = new Yaml()
        List yamls = []
        for (String yaml : modification.applyYaml.split("---")) {
            yamls.add(parser.load(yaml))
        }
        yamls.each {
            assert appliedNetworkPolicies.get(it."metadata"."namespace")?.contains(it."metadata"."name")
        }

        switch (deleteMode) {
            case DeleteExistingPoliciesMode.ALL:
                assert modification.toDeleteList.findAll {
                    it.name == policy1.name || it.name == policy2.name
                }.size() == 2
                preExistingNetworkPolicies.each { k, v ->
                    v.each {
                        assert !appliedNetworkPolicies.get(k).contains(it)
                    }
                }
                break
            case DeleteExistingPoliciesMode.NONE:
                assert modification.toDeleteCount == 0
                assert !yamls.find { it."metadata"."name" == "stackrox-generated-${NGINXCONNECTIONTARGET}" }
                preExistingNetworkPolicies.each { k, v ->
                    v.each {
                        assert appliedNetworkPolicies.get(k).contains(it)
                    }
                }
                break
            case DeleteExistingPoliciesMode.GENERATED_ONLY:
                assert modification.toDeleteList.find { it.name == policy2.name }
                assert !yamls.find { it."metadata"."name" == "stackrox-generated-${NGINXCONNECTIONTARGET}" }
                preExistingNetworkPolicies.each { k, v ->
                    v.each {
                        if (it.startsWith("generated-")) {
                            assert !appliedNetworkPolicies.get(k).contains(it)
                        } else {
                            assert appliedNetworkPolicies.get(k).contains(it)
                        }
                    }
                }
                break
        }

        and:
        "Undo applied policies and verify orchestrator state goes back to pre-existing state"
        def undoRecord = NetworkPolicyService.undoGeneratedNetworkPolicy()
        assert undoRecord.originalModification == modification

        assert !(
                NetworkPolicyService.applyGeneratedNetworkPolicy(undoRecord.undoModification)
                        instanceof StatusRuntimeException
        )
        def undoNetworkPolicies = getQANetworkPoliciesNamesByNamespace(true)
        log.info "${undoNetworkPolicies}"
        assert undoNetworkPolicies == preExistingNetworkPolicies

        cleanup:
        "remove policies"
        policyId1 ? orchestrator.deleteNetworkPolicy(policy1) : null
        policyId2 ? orchestrator.deleteNetworkPolicy(policy2) : null

        where:
        "data inputs:"
        deleteMode | note
        DeleteExistingPoliciesMode.NONE | ""

        // Run same tests a second time to make sure we can apply -> undo -> apply again
        DeleteExistingPoliciesMode.NONE | "(repeat)"

        DeleteExistingPoliciesMode.GENERATED_ONLY | ""
        DeleteExistingPoliciesMode.ALL | ""
    }

    @Tag("BAT")
    @Tag("NetworkFlowVisualization")
    @Ignore("Skip this test until we can determine a more reliable way to test")
    def "Apply a generated network policy and verify connection states"() {
        given:
        "Initial graph state and existing network policies"
        NetworkGraph baseGraph = NetworkGraphService.getNetworkGraph()

        and:
        "Get generated network policies"
        def modification = NetworkPolicyService.generateNetworkPolicies()

        when:
        "We can apply generated network policies to an environment"
        NetworkPolicyService.applyGeneratedNetworkPolicy(modification)

        and:
        "let netpols propagate and allow connection data to update, then verify graph again"
        sleep 60000
        NetworkGraph newGraph = NetworkGraphService.getNetworkGraph()
        for (NetworkNode newNode : newGraph.nodesList) {
            def baseNode = baseGraph.nodesList.find {
                it.entity.deployment.name == newNode.entity.deployment.name &&
                        it.entity.deployment.namespace == newNode.entity.deployment.namespace
            }
            assert newNode.outEdgesMap.keySet().sort() == baseNode.outEdgesMap.keySet().sort()
        }

        then:
        "Undo applied policies"
        NetworkPolicyService.applyGeneratedNetworkPolicy(
                NetworkPolicyService.undoGeneratedNetworkPolicy().undoModification)
    }

    private static getNode(String deploymentId, boolean withListenPorts, int timeoutSeconds = 90) {
        def t = new Timer(timeoutSeconds, 1)

        NetworkNode match = null
        while (t.IsValid()) {
            def graph = NetworkGraphService.getNetworkGraph()
            def node = NetworkGraphUtil.findDeploymentNode(graph, deploymentId)
            if (node) {
                match = node
            }
            if (!node || (withListenPorts && !node?.entity?.deployment?.listenPortsCount)) {
                continue
            }
            return node
        }

        return match
    }

    private waitForEdgeToBeClosed(Edge edge, int timeoutSeconds = 65) {
        int intervalSeconds = 1
        int waitTime
        def prevEdge = edge
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            def graph = NetworkGraphService.getNetworkGraph()
            def newEdge = NetworkGraphUtil.findEdges(graph, edge.sourceID, edge.targetID)?.find { true }

            // If lastActiveTimestamp is equal to the previous edges lastActiveTimestamp then the edge has been closed
            if (newEdge != null && newEdge.lastActiveTimestamp == prevEdge.lastActiveTimestamp) {
                return true
            }
            prevEdge = newEdge
            sleep intervalSeconds * 1000
        }
        log.info "Edge was never closed"
        return false
    }

    private waitForEdgeUpdate(Edge edge, int timeoutSeconds = 60, float addSecondsToEdgeTimestamp = 0.2) {
        int intervalSeconds = 1
        int waitTime
        def startTime = System.currentTimeMillis()
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            def graph = NetworkGraphService.getNetworkGraph()
            def newEdge = NetworkGraphUtil.findEdges(graph, edge.sourceID, edge.targetID)?.find { true }

            // Added an optional buffer here with addSecondsToEdgeTimestamp. Test was flakey
            // because we cannot guarantee when an edge will stop appearing in the data pipeline
            // the buffer simply says only check for updates that happen >`addSecondsToEdgeTimestamp`
            // seconds after the baseline edge.
            // In addition per ROX-5749 small deltas may appear in edge timestamps which will not be
            // considered as a new edge, hence the 0.2 default value.
            if (newEdge != null &&
                    newEdge.lastActiveTimestamp > edge.lastActiveTimestamp + (addSecondsToEdgeTimestamp * 1000)) {
                log.info "Found updated edge in graph after ${(System.currentTimeMillis() - startTime) / 1000}s"
                log.info "The updated edge is " +
                        "${((newEdge.lastActiveTimestamp - edge.lastActiveTimestamp)/1000) as Integer} " +
                        "seconds later"
                return newEdge
            }
            sleep intervalSeconds * 1000
        }
        log.info "SR did not detect updated edge in Network Flow graph"
        return null
    }

    def getQANetworkPoliciesNamesByNamespace(boolean ignoreUndoneStackroxGenerated) {
        return orchestrator.getAllNetworkPoliciesNamesByNamespace(ignoreUndoneStackroxGenerated).findAll {
            it.key.startsWith("qa")
        }
    }
}
