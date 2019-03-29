import static com.jayway.restassured.RestAssured.given

import orchestratormanager.OrchestratorTypes
import util.Env
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.NetworkGraphOuterClass
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass.GenerateNetworkPoliciesRequest.DeleteExistingPoliciesMode
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicyModification
import org.yaml.snakeyaml.Yaml
import services.NetworkPolicyService
import com.google.protobuf.Timestamp
import groups.BAT
import groups.RUNTIME
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
import io.stackrox.proto.storage.NetworkFlowOuterClass.L4Protocol
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntityInfo.Type
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
                                      "do echo \"Hello from ${UDPCONNECTIONSOURCE}\" | " +
                                      "socat -d -d -d -d -s STDIN UDP:${UDPCONNECTIONTARGET}:8080; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(TCPCONNECTIONSOURCE)
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addLabel("app", TCPCONNECTIONSOURCE)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${TCPCONNECTIONSOURCE}\" | " +
                                      "socat -d -d -d -d -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "done" as String,]),
            new Deployment()
                    .setName(MULTIPLEPORTSCONNECTION)
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addLabel("app", MULTIPLEPORTSCONNECTION)
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                      "do echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
                                      "socat -s STDIN TCP:${TCPCONNECTIONTARGET}:80; " +
                                      "echo \"Hello from ${MULTIPLEPORTSCONNECTION}\" | " +
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
            new Deployment()
                    .setName("${TCPCONNECTIONSOURCE}-qa2")
                    .setNamespace("qa2")
                    .setImage("apollo-dtr.rox.systems/qa/socat:testing")
                    .addLabel("app", "${TCPCONNECTIONSOURCE}-qa2")
                    .setCommand(["/bin/sh", "-c",])
                    .setArgs(["while sleep 5; " +
                                  "do echo \"Hello from ${TCPCONNECTIONSOURCE}-qa2\" | " +
                                  "socat -d -d -d -d -s STDIN TCP:${TCPCONNECTIONTARGET}.qa.svc.cluster.local:80; " +
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
    @Category([BAT, RUNTIME, NetworkFlowVisualization])
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
        UDPCONNECTIONSOURCE  | UDPCONNECTIONTARGET   | L4Protocol.L4_PROTOCOL_UDP
        TCPCONNECTIONSOURCE  | TCPCONNECTIONTARGET   | L4Protocol.L4_PROTOCOL_TCP
        //ICMPCONNECTIONSOURCE | NGINXCONNECTIONTARGET | L4Protocol.L4_PROTOCOL_ICMP
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
        "Check for edge in network graph"
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
        List<Edge> edges = checkForEdge("", deploymentUid, null, 180)
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

    @Category([BAT])
    def "Verify generated network policies"() {
        given:
        "Get current state of network graph"
        NetworkGraph currentGraph = NetworkGraphService.getNetworkGraph()
        List<String> deployedNamespaces = DEPLOYMENTS*.namespace

        and:
        "delete a deployment"
        Deployment delete = DEPLOYMENTS.find { it.name == NOCONNECTIONSOURCE }
        orchestrator.deleteDeployment(delete)
        Services.waitForSRDeletion(delete)

        when:
        "Generate Network Policies"
        NetworkPolicyModification modification = NetworkPolicyService.generateNetworkPolicies()
        Yaml parser = new Yaml()
        List yamls = []
        for (String yaml : modification.applyYaml.split("---")) {
            yamls.add(parser.load(yaml))
        }

        then:
        "verify generated netpols vs current graph state"
        yamls.each {
            assert it."metadata"."namespace" != "kube-system" &&
                    it."metadata"."namespace" != "kube-public"
        }
        yamls.findAll {
            deployedNamespaces.contains(it."metadata"."namespace")
        }.each {
            String deploymentName =
                    it."metadata"."name"["stackrox-generated-".length()..it."metadata"."name".length() - 1]
            assert deploymentName != NOCONNECTIONSOURCE
            assert it."metadata"."labels"."network-policy-generator.stackrox.io/generated"
            assert it."metadata"."namespace"
            def index = currentGraph.nodesList.findIndexOf { node -> node.deploymentName == deploymentName }
            def allowAllIngress = DEPLOYMENTS.find { it.name == deploymentName }?.createLoadBalancer ||
                    currentGraph.nodesList.find { it.entity.type == Type.INTERNET }.outEdgesMap.containsKey(index)
            List<NetworkGraphOuterClass.NetworkNode> outNodes =  currentGraph.nodesList.findAll { node ->
                node.outEdgesMap.containsKey(index)
            }
            def ingressPodSelectors = it."spec"."ingress".find { it.containsKey("from") } ?
                    it."spec"."ingress".get(0)."from".findAll { it.containsKey("podSelector") } :
                    null
            def ingressNamespaceSelectors = it."spec"."ingress".find { it.containsKey("from") } ?
                    it."spec"."ingress".get(0)."from".findAll { it.containsKey("namespaceSelector") } :
                    null

            if (allowAllIngress) {
                print "${deploymentName} has LB/External incoming traffic - ensure All Ingress allowed"
                assert it."spec"."ingress" == [[:]]
            } else if (outNodes.size() > 0) {
                print "${deploymentName} has incoming connections - ensure podSelectors/namespaceSelectors match " +
                        "sources from graph"
                def sourceDeploymentsFromGraph = outNodes.findAll { it.deploymentName }*.deploymentName
                def sourceDeploymentsFromNetworkPolicy = ingressPodSelectors.collect {
                    it."podSelector"."matchLabels"."app"
                }
                def sourceNamespacesFromNetworkPolicy = ingressNamespaceSelectors.collect {
                    it."namespaceSelector"."matchLabels"."namespace.metadata.stackrox.io/name"
                }
                assert sourceDeploymentsFromNetworkPolicy.sort() == sourceDeploymentsFromGraph.sort()
                assert deployedNamespaces.containsAll(sourceNamespacesFromNetworkPolicy)
            } else {
                print "${deploymentName} has no incoming connections - ensure ingress spec is empty"
                assert it."spec"."ingress" == [] || it."spec"."ingress" == null
            }
        }
    }

    @Unroll
    @Category([BAT])
    def "Verify network policy generator apply/undo with delete modes: #deleteMode"() {
        //skip on OS for now
        Assume.assumeTrue(Env.mustGetOrchestratorType() == OrchestratorTypes.K8S)

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
        def preExistingNetworkPolicies = orchestrator.getAllNetworkPoliciesNamesByNamespace(true)
        println preExistingNetworkPolicies

        expect:
        "actual policies should exist in generated response depending on delete mode"
        def modification = NetworkPolicyService.generateNetworkPolicies(deleteMode)
        assert !(NetworkPolicyService.applyGeneratedNetworkPolicy(modification) instanceof StatusRuntimeException)
        def appliedNetworkPolicies = orchestrator.getAllNetworkPoliciesNamesByNamespace(true)
        println appliedNetworkPolicies

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
        def undoNetworkPolicies = orchestrator.getAllNetworkPoliciesNamesByNamespace(true)
        println undoNetworkPolicies
        assert undoNetworkPolicies == preExistingNetworkPolicies

        cleanup:
        "remove policies"
        policyId1 ? orchestrator.deleteNetworkPolicy(policy1) : null
        policyId2 ? orchestrator.deleteNetworkPolicy(policy2) : null

        where:
        "data inputs:"
        deleteMode | _
        DeleteExistingPoliciesMode.NONE | _

        // Run same tests a second time to make sure we can apply -> undo -> apply again
        DeleteExistingPoliciesMode.NONE | _

        DeleteExistingPoliciesMode.GENERATED_ONLY | _
        DeleteExistingPoliciesMode.ALL | _
    }

    @Category([BAT, NetworkFlowVisualization])
    def "Apply a generated network policy and verify connection states"() {
        // Skip this test until we can determine a more reliable way to test
        Assume.assumeTrue(false)

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
        for (NetworkGraphOuterClass.NetworkNode newNode : newGraph.nodesList) {
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
