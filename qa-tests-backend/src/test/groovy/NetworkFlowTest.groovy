import static com.jayway.restassured.RestAssured.given

import orchestratormanager.OrchestratorTypes
import com.google.protobuf.Timestamp
import groups.BAT
import groups.NetworkFlowVisualization
import objects.Deployment
import objects.Edge
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.NetworkGraphService
import spock.lang.Unroll
import util.NetworkGraphUtil
import io.stackrox.proto.storage.NetworkFlowOuterClass
import io.stackrox.proto.api.v1.NetworkGraphOuterClass.NetworkGraph
import com.google.protobuf.util.Timestamps

class NetworkFlowTest extends BaseSpecification {

    static final private NETWORK_FLOW_UPDATE_CADENCE = 30000 // Network flow data is updated every 30 seconds

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

    static final private List<Deployment> DEPLOYMENTS = [
            //Target deployments
            new Deployment()
                    .setName(UDPCONNECTIONTARGET)
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addPort(8080, "UDP")
                    .addLabel("app", UDPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["socat -d -d -v UDP-RECV:8080 STDOUT",]),
            new Deployment()
                    .setName(TCPCONNECTIONTARGET)
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addPort(80)
                    .addPort(8080)
                    .addLabel("app", TCPCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["(socat -d -d -v TCP-LISTEN:80,fork STDOUT & " +
                                      "socat -d -d -v TCP-LISTEN:8080,fork STDOUT)" as String,]),
            new Deployment()
                    .setName(NGINXCONNECTIONTARGET)
                    .setImage("nginx")
                    .addPort(80)
                    .addLabel("app", NGINXCONNECTIONTARGET)
                    .setExposeAsService(true)
                    .setCreateLoadBalancer(true),

            //Source deployments
            new Deployment()
                    .setName(NOCONNECTIONSOURCE)
                    .setImage("nginx")
                    .addLabel("app", NOCONNECTIONSOURCE),
            new Deployment()
                    .setName(SHORTCONSISTENTSOURCE)
                    .setImage("nginx:1.15.4-alpine")
                    .addLabel("app", SHORTCONSISTENTSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NETWORK_FLOW_UPDATE_CADENCE / 1000}; " +
                                      "do wget -S http://${NGINXCONNECTIONTARGET}; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(SINGLECONNECTIONSOURCE)
                    .setImage("nginx:1.15.4-alpine")
                    .addLabel("app", SINGLECONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["wget -S -T 2 http://${NGINXCONNECTIONTARGET} && " +
                                      "while sleep 30; do echo hello; done" as String,]),
            new Deployment()
                    .setName(UDPCONNECTIONSOURCE)
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addLabel("app", UDPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do socat -d -d -d -d -s STDIN UDP:${UDPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONSOURCE)
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addLabel("app", TCPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do socat -d -d -d -d -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(MULTIPLEPORTSCONNECTION)
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addLabel("app", MULTIPLEPORTSCONNECTION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do socat -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "socat -s STDIN TCP:${TCPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(EXTERNALDESTINATION)
                    .setImage("nginx:1.15.4-alpine")
                    .addLabel("app", EXTERNALDESTINATION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep ${NETWORK_FLOW_UPDATE_CADENCE / 1000}; " +
                                      "do wget -S http://www.google.com; " +
                                      "done" as String,]),
    ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
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
        for (Deployment d : DEPLOYMENTS) {
            assert Services.waitForDeployment(d)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Unroll
    @Category([BAT, NetworkFlowVisualization])
    def "Verify connections can be detected: #protocol"() {
        given:
        "Two deployments, A and B, where B communicates to A via #protocol"
        String targetUid = DEPLOYMENTS.find { it.name == targetDeployment }?.deploymentUid
        assert targetUid != null
        String sourceUid = DEPLOYMENTS.find { it.name == sourceDeployment }?.deploymentUid
        assert sourceUid != null

        expect:
        "Check for edge in network graph"
        println "Checking for edge between ${sourceDeployment} and ${targetDeployment}"
        List<Edge> edges = checkForEdge(sourceUid, targetUid)

        // Due to flakey tests, adding some debugging logging in the event that we don't find the expected edge
        if (edges == null) {
            println "*** SOURCE LOGS ***\n" +
                    orchestrator.getContainerlogs(DEPLOYMENTS.find { it.name == sourceDeployment })
            println "*** TARGET LOGS ***\n" +
                    orchestrator.getContainerlogs(DEPLOYMENTS.find { it.name == targetDeployment })
            println "*** NETWORK GRAPH ***\n" +
                    NetworkGraphService.getNetworkGraph()
        }
        assert edges
        assert edges.get(0).protocol == protocol
        assert DEPLOYMENTS.find { it.name == targetDeployment }?.ports?.keySet()?.contains(edges.get(0).port)

        where:
        "Data is:"

        sourceDeployment     | targetDeployment      | protocol
        UDPCONNECTIONSOURCE  | UDPCONNECTIONTARGET   | NetworkFlowOuterClass.L4Protocol.L4_PROTOCOL_UDP
        TCPCONNECTIONSOURCE  | TCPCONNECTIONTARGET   | NetworkFlowOuterClass.L4Protocol.L4_PROTOCOL_TCP
        //ICMPCONNECTIONSOURCE | NGINXCONNECTIONTARGET | NetworkFlowOuterClass.L4Protocol.L4_PROTOCOL_ICMP
    }

    @Category([NetworkFlowVisualization])
    def "Verify connections with short consistent intervals between 2 deployments"() {
        given:
        "Two deployments, A and B, where B communicates to A in short consistent intervals"
        String targetUid = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = DEPLOYMENTS.find { it.name == SHORTCONSISTENTSOURCE }?.deploymentUid
        assert sourceUid != null

        when:
        "Check for edge in entwork graph"
        println "Checking for edge between ${SHORTCONSISTENTSOURCE} and ${NGINXCONNECTIONTARGET}"
        List<Edge> edges = checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "Wait for collector update and fetch graph again to confirm short interval connections remain"
        assert waitForEdgeUpdate(edges.get(0), 90)
    }

    @Category([NetworkFlowVisualization])
    def "Verify connections to external sources"() {
        given:
        "Deployment A, where A communicates to an external target"
        String deploymentUid = DEPLOYMENTS.find { it.name == EXTERNALDESTINATION }?.deploymentUid
        assert deploymentUid != null

        expect:
        "Check for edge in network graph"
        println "Checking for edge from ${EXTERNALDESTINATION} to external target"
        List<Edge> edges = checkForEdge(deploymentUid, "")
        assert edges
    }

    @Category([NetworkFlowVisualization])
    def "Verify connections from external sources"() {
        given:
        "Cluster is Openshift"
        // This is due to ROX-897, external -> deploy in k8s via LB is not working
        Assume.assumeTrue(OrchestratorTypes.valueOf(System.getenv("CLUSTER")) == OrchestratorTypes.OPENSHIFT)

        and:
        "Deployment A, where an external source communicates to A"
        String deploymentUid = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert deploymentUid != null
        String deploymentIP = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.loadBalancerIP
        assert deploymentIP != null

        when:
        "ping the target deployment"
        def response = given().get("http://${deploymentIP}")
        println response.asString()

        then:
        "Check for edge in network graph"
        println "Checking for edge from external target to ${EXTERNALDESTINATION}"
        List<Edge> edges = checkForEdge("", deploymentUid)
        assert edges
    }

    @Category([NetworkFlowVisualization])
    def "Verify no connections between 2 deployments"() {
        given:
        "Two deployments, A and B, where neither communicates to the other"
        String targetUid = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = DEPLOYMENTS.find { it.name == NOCONNECTIONSOURCE }?.deploymentUid
        assert sourceUid != null

        expect:
        "Assert connection states"
        println "Checking for NO edge between ${NOCONNECTIONSOURCE} and ${NGINXCONNECTIONTARGET}"
        assert !checkForEdge(sourceUid, targetUid, null, 30)
    }

    @Category([NetworkFlowVisualization])
    def "Verify one-time connections show at first, but do not appear again"() {
        given:
        "Two deployments, A and B, where B communicates to A a single time during initial deployment"
        String targetUid = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = DEPLOYMENTS.find { it.name == SINGLECONNECTIONSOURCE }?.deploymentUid
        assert sourceUid != null

        when:
        "Check for edge in entwork graph"
        println "Checking for edge between ${SINGLECONNECTIONSOURCE} and ${NGINXCONNECTIONTARGET}"
        List<Edge> edges = checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "Wait for collector update and fetch graph again to confirm connection dropped"
        assert !waitForEdgeUpdate(edges.get(0))
    }

    @Category([NetworkFlowVisualization])
    def "Verify connections between two deployments on 2 separate ports shows both edges in the graph"() {
        given:
        "Two deployments, A and B, where B communicates to A on 2 different ports"
        String targetUid = DEPLOYMENTS.find { it.name == TCPCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = DEPLOYMENTS.find { it.name == MULTIPLEPORTSCONNECTION }?.deploymentUid
        assert sourceUid != null

        when:
        "Check for edge in entwork graph"
        List<Edge> edges = checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "Assert that there are 2 connection edges"
        assert edges.size() == 2
    }

    @Category([NetworkFlowVisualization])
    def "Verify cluster updates can block flow connections from showing"() {
        given:
        "orchestrator supports NetworkPolicies"
        // limit this test to run only on environments that support Network Policies
        Assume.assumeTrue(orchestrator.supportsNetworkPolicies())

        and:
        "Two deployments, A and B, where B communicates to A"
        String targetUid = DEPLOYMENTS.find { it.name == NGINXCONNECTIONTARGET }?.deploymentUid
        assert targetUid != null
        String sourceUid = DEPLOYMENTS.find { it.name == SHORTCONSISTENTSOURCE }?.deploymentUid
        assert sourceUid != null

        when:
        "apply network policy to block ingress to A"
        NetworkPolicy policy = new NetworkPolicy("deny-all-traffic-to-a")
                .setNamespace("qa")
                .addPodSelector(["app":NGINXCONNECTIONTARGET])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy)
        println "Sleeping 60s to allow policy to propagate and flows to update after propagation"
        sleep 60000

        and:
        "Check for original edge in network graph"
        println "Checking for edge between ${SHORTCONSISTENTSOURCE} and ${NGINXCONNECTIONTARGET}"
        List<Edge> edges = checkForEdge(sourceUid, targetUid)
        assert edges

        then:
        "make sure edge does not get updated"
        //Use a 20 second buffer to account for additional edges coming in through the data pipeline
        assert !waitForEdgeUpdate(edges.get(0), 60, 20)

        cleanup:
        "remove policy"
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy)
        }
    }

    @Category([NetworkFlowVisualization])
    def "Verify edge timestamps are never in the future, or before start of flow tests"() {
        given:
        "Get current state of edges and current timestamp"
        def queryString = "Deployment:" + DEPLOYMENTS.name.join(",")
        NetworkGraph currentGraph = NetworkGraphService.getNetworkGraph(null, queryString)
        long currentTime = System.currentTimeMillis()

        expect:
        "Check timestamp for each edge"
        for (Edge edge : NetworkGraphUtil.findEdges(currentGraph, null, null)) {
            assert edge.lastActiveTimestamp <= currentTime + 2000 //allow up to 2 sec leeway
            assert edge.lastActiveTimestamp >= Timestamps.toMillis(testStartTime)
        }
    }

    private checkForEdge(String sourceId, String targetId, Timestamp since = null, int timeoutSeconds = 90) {
        int intervalSeconds = 1
        int waitTime
        def startTime = System.currentTimeMillis()
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            if (waitTime > 0) {
                sleep intervalSeconds * 1000
            }

            def graph = NetworkGraphService.getNetworkGraph(since)
            def edges = NetworkGraphUtil.findEdges(graph, sourceId, targetId)
            if (edges != null && edges.size() > 0) {
                println "Found source -> target in graph after ${(System.currentTimeMillis() - startTime) / 1000}s"
                return edges
            }
        }
        println "SR did not detect the edge in Network Flow graph"
        return null
    }

    private waitForEdgeUpdate(Edge edge, int timeoutSeconds = 60, int addSecondsToEdgeTimestamp = 0) {
        int intervalSeconds = 1
        int waitTime
        def startTime = System.currentTimeMillis()
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            def graph = NetworkGraphService.getNetworkGraph()
            def newEdge = NetworkGraphUtil.findEdges(graph, edge.sourceID, edge.targetID)?.find { true }

            // Added an optional buffer here with addSecondsToEdgeTimestamp. Test was flakey
            // because we cannot guarantee when an edge will stop appearing in the data pipeline
            // the buffer simply says only check for updates that happen >`addSecondsToEdgeTimestamp`
            // seconds after the baseline edge
            if (newEdge != null &&
                    newEdge.lastActiveTimestamp > edge.lastActiveTimestamp + (addSecondsToEdgeTimestamp * 1000)) {
                println "Found updated edge in graph after ${(System.currentTimeMillis() - startTime) / 1000}s"
                return newEdge
            }
            sleep intervalSeconds * 1000
        }
        println "SR did not detect updated edge in Network Flow graph"
        return null
    }
}
