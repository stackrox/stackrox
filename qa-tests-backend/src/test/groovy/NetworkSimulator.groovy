import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass
import io.stackrox.proto.storage.NetworkPolicyOuterClass.NetworkPolicyReference

import common.Constants
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.SlackNotifier
import services.NetworkGraphService
import services.NetworkPolicyService
import util.Env
import util.NetworkGraphUtil

import spock.lang.IgnoreIf
import spock.lang.Tag
import spock.lang.Unroll

class NetworkSimulator extends BaseSpecification {

    // Deployment names
    static final private String WEBDEPLOYMENT = "web"
    static final private String WEB2DEPLOYMENT = "alt-web"
    static final private String CLIENTDEPLOYMENT = "client"
    static final private String CLIENT2DEPLOYMENT = "alt-client"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName(WEBDEPLOYMENT)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addPort(80)
                    .addLabel("app", WEBDEPLOYMENT),
            new Deployment()
                    .setName(WEB2DEPLOYMENT)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addLabel("app", WEB2DEPLOYMENT),
            new Deployment()
                    .setName(CLIENTDEPLOYMENT)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addPort(443)
                    .addLabel("app", CLIENTDEPLOYMENT),
            new Deployment()
                    .setName(CLIENT2DEPLOYMENT)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                    .addLabel("app", CLIENT2DEPLOYMENT),
    ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        DEPLOYMENTS.each { Services.waitForDeployment(it) }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Tag("NetworkPolicySimulation")
    @Tag("BAT")
    
    def "Verify NetworkPolicy Simulator replace existing network policy"() {
        given:
        def allDeps = NetworkGraphUtil.getDeploymentsAsGraphNodes()

        when:
        "apply network policy"
        NetworkPolicy policy = new NetworkPolicy("deny-all-namespace-ingress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        orchestrator.applyNetworkPolicy(policy)
        assert NetworkPolicyService.waitForNetworkPolicy(policy.uid)
        def baseline = NetworkPolicyService.getNetworkPolicyGraph(null, scope)

        and:
        "generate simulation"
        policy.addPolicyType(NetworkPolicyTypes.EGRESS)
        def policyYAML = orchestrator.generateYaml(policy)
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(policyYAML, null, scope)
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() ==
                NetworkGraphUtil.findEdges(baseline, null, webAppId).size()
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert simulation.policiesList.find { it.policy.name == "deny-all-namespace-ingress" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.MODIFIED

        assert NetworkGraphUtil.verifyGraphFilterAndScope(simulation.simulatedGraph, allDeps.nonOrchestratorDeployments,
                allDeps.orchestratorDeployments, true, orchestratorDepsShouldExist)

        // Verify no change to added nodes
        assert simulation.added.nodeDiffsCount == 0
        // Verify old deployment node has outedges
        assert simulation.removed.nodeDiffsMap.get(webAppId).outEdgesCount > 0

        def nonQAOutEdges = baseline.nodesList.collectEntries {
            if (it.namespace != "qa" || !it.deploymentId) {
                return Collections.emptyMap()
            }
            return [it.deploymentId, it.outEdgesMap.keySet().count {
                baseline.nodesList.get(it).namespace != "qa"
            },]
        }

        simulation.removed.nodeDiffsMap.each { k, v ->
            def node = simulation.simulatedGraph.nodesList.find { it.deploymentId == k }

            assert node.namespace == "qa"
            assert v.policyIdsCount == 0
            assert v.outEdgesMap.size() == nonQAOutEdges.get(node.deploymentId)
            assert v.getNonIsolatedEgress()
        }
        cleanup:
        "cleanup"
        Services.cleanupNetworkPolicies([policy])

        where:
        "Data is:"

        scope                           | orchestratorDepsShouldExist
        "Orchestrator Component:false"  | false
        ""                              | true
    }

    @Tag("NetworkPolicySimulation")
    @Tag("BAT")
    
    def "Verify NetworkPolicy Simulator add to an existing network policy"() {
        given:
        def allDeps = NetworkGraphUtil.getDeploymentsAsGraphNodes()

        when:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        orchestrator.applyNetworkPolicy(policy1)
        assert NetworkPolicyService.waitForNetworkPolicy(policy1.uid)
        def baseline = NetworkPolicyService.getNetworkPolicyGraph(null, scope)

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(orchestrator.generateYaml(policy2), null,
                scope)
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def clientAppId = simulation.simulatedGraph.nodesList.find {
            it.deploymentName == CLIENTDEPLOYMENT
        }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.verifyGraphFilterAndScope(simulation.simulatedGraph, allDeps.nonOrchestratorDeployments,
                allDeps.orchestratorDeployments, true, orchestratorDepsShouldExist)

        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() > 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, clientAppId).size() ==
                NetworkGraphUtil.findEdges(baseline, null, clientAppId).size()
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() ==
                NetworkGraphUtil.findEdges(baseline, webAppId, null).size()
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, clientAppId, null).size() ==
                NetworkGraphUtil.findEdges(baseline, clientAppId, null).size()

        assert simulation.policiesList.find { it.policy.name == "deny-all-traffic" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED

        assert simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        // Verify outEdge to 'web' is added to all nodes
        simulation.added.nodeDiffsMap.each {
            if (it.key != webAppId) {
                assert it.value.outEdgesMap.containsKey(webAppId)
            } else {
                assert it.value.policyIdsCount > 0
            }
        }

        // Verify removed is not changed
        assert simulation.removed.nodeDiffsMap.size() == 0

        cleanup:
        "cleanup"
        Services.cleanupNetworkPolicies([policy1])

        where:
        "Data is:"

        scope                           | orchestratorDepsShouldExist
        "Orchestrator Component:false"  | false
        ""                              | true
    }

    @Tag("NetworkPolicySimulation")
    @Tag("BAT")
    
    def "Verify NetworkPolicy Simulator with query - multiple policy simulation"() {
        given:
        def allDeps = new NetworkGraphUtil().getDeploymentsAsGraphNodes()

        when:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        orchestrator.applyNetworkPolicy(policy1)
        assert NetworkPolicyService.waitForNetworkPolicy(policy1.uid)

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressPodSelector(["app": CLIENTDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        NetworkPolicy policy3 = new NetworkPolicy("allow-egress-application-client")
                .setNamespace("qa")
                .addPodSelector(["app": CLIENTDEPLOYMENT])
                .addEgressPodSelector(["app": WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy2) + orchestrator.generateYaml(policy3),
                "Deployment:\"web\",\"client\"+Namespace:qa", scope)
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def clientAppId = simulation.simulatedGraph.nodesList.find {
            it.deploymentName == CLIENTDEPLOYMENT
        }.deploymentId
        def numNonIsolatedEgressNodes = simulation.simulatedGraph.nodesList.count { it.nonIsolatedEgress }
        def numNonIsolatedIngressNodes = simulation.simulatedGraph.nodesList.count { it.nonIsolatedIngress }

        then:
        "verify simulation"
        assert NetworkGraphUtil.verifyGraphFilterAndScope(simulation.simulatedGraph, allDeps.nonOrchestratorDeployments,
                allDeps.orchestratorDeployments, true, orchestratorDepsShouldExist)

        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() >=
                numNonIsolatedEgressNodes

        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, clientAppId).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, clientAppId, null).size() >=
                numNonIsolatedIngressNodes

        // No connections from INTERNET to "qa" namespace; simulated graph is scoped to "qa" namespace.
        assert NetworkGraphUtil.findEdges(
                simulation.simulatedGraph,
                Constants.INTERNET_EXTERNAL_SOURCE_ID,
                null).size() == 0

        assert simulation.simulatedGraph.nodesList.size() == 2

        assert simulation.policiesList.find { it.policy.name == "deny-all-traffic" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED

        assert simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        assert simulation.policiesList.find { it.policy.name == "allow-egress-application-client" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        // Verify outEdge to 'web' is added to 'client' node only
        simulation.added.nodeDiffsMap.each {
            if (it.key == clientAppId) {
                assert it.value.outEdgesMap.containsKey(webAppId)
                assert it.value.outEdgesCount == 1
            } else {
                assert it.value.policyIdsCount > 0
                assert !it.value.outEdgesMap.containsKey(webAppId)
            }
        }

        cleanup:
        "cleanup"
        Services.cleanupNetworkPolicies([policy1])

        where:
        "Data is:"

        scope                           | orchestratorDepsShouldExist
        "Orchestrator Component:false"  | false
        // although unscoped, the net pol allows connection only between non-orchestrator components,
        // hence no orchestrator components are expected in simulation.
        ""                              | false
    }

    @Tag("NetworkPolicySimulation")
    @Tag("BAT")
    
    // skip if executed in a test environment with just secured-cluster deployed in the test cluster
    // i.e. central is deployed elsewhere
    @IgnoreIf({ Env.ONLY_SECURED_CLUSTER == "true" })
    def "Verify NetworkPolicy Simulator with query - single policy simulation"() {
        given:
        def allDeps = NetworkGraphUtil.getDeploymentsAsGraphNodes()

        when:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        orchestrator.applyNetworkPolicy(policy1)
        assert NetworkPolicyService.waitForNetworkPolicy(policy1.uid)

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(orchestrator.generateYaml(policy2),
                "Deployment:\"web\"+Namespace:qa", scope)
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        // Ensure that central is present
        def centralAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == "central" }.deploymentId
        def numNonIsolatedEgressNodes = simulation.simulatedGraph.nodesList.count { it.nonIsolatedEgress }

        then:
        "verify simulation"
        assert NetworkGraphUtil.verifyGraphFilterAndScope(simulation.simulatedGraph, allDeps.nonOrchestratorDeployments,
                allDeps.orchestratorDeployments, true, orchestratorDepsShouldExist)

        // At least nodes with non-isolated egress should have edges to 'web' deployment.
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() >=
                numNonIsolatedEgressNodes
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        // Verify that central is present as a peer if when not queried.
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, centralAppId, null).size() == 1

        assert simulation.simulatedGraph.nodesList.size() > 3 // central should now be part of peers without querying it

        assert simulation.policiesList.find { it.policy.name == "deny-all-traffic" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED

        assert simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        // Verify outEdge to 'web' is added to all nodes
        simulation.added.nodeDiffsMap.each {
            if (it.key != webAppId) {
                assert it.value.outEdgesMap.containsKey(webAppId)
            } else {
                assert it.value.policyIdsCount > 0
            }
        }

        // Verify removed is not changed
        assert simulation.removed.nodeDiffsMap.size() == 0

        cleanup:
        "cleanup"
        Services.cleanupNetworkPolicies([policy1])

        where:
        "Data is:"

        scope                           | orchestratorDepsShouldExist
        "Orchestrator Component:false"  | false
        ""                              | true
    }

    @Tag("NetworkPolicySimulation")
    @Tag("BAT")
    
    def "Verify NetworkPolicy Simulator with delete policies"() {
        given:
        def allDeps = NetworkGraphUtil.getDeploymentsAsGraphNodes()

        when:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        orchestrator.applyNetworkPolicy(policy1)
        assert NetworkPolicyService.waitForNetworkPolicy(policy1.uid)
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        orchestrator.applyNetworkPolicy(policy2)
        assert NetworkPolicyService.waitForNetworkPolicy(policy2.uid)
        def baseline = NetworkGraphService.getNetworkGraph(null, scope)

        and:
        "compile list of to delete policies"
        def toDelete = [
                NetworkPolicyReference.newBuilder()
                        .setName("allow-ingress-application-web")
                        .setNamespace("qa")
                        .build(),
        ]

        and:
        "generate simulation"
        NetworkPolicy policy3 = new NetworkPolicy("allow-ingress-application-client")
                .setNamespace("qa")
                .addPodSelector(["app": CLIENTDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(orchestrator.generateYaml(policy3), null,
                scope, toDelete)
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def clientAppId = simulation.simulatedGraph.nodesList.find {
            it.deploymentName == CLIENTDEPLOYMENT
        }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.verifyGraphFilterAndScope(simulation.simulatedGraph, allDeps.nonOrchestratorDeployments,
                allDeps.orchestratorDeployments, true, orchestratorDepsShouldExist)

        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, clientAppId).size() > 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() ==
                NetworkGraphUtil.findEdges(baseline, null, webAppId).size()
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() ==
                NetworkGraphUtil.findEdges(baseline, webAppId, null).size()
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, clientAppId, null).size() ==
                NetworkGraphUtil.findEdges(baseline, clientAppId, null).size()

        assert simulation.policiesList.find { it.policy.name == "deny-all-traffic" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED

        assert simulation.policiesList.find { it.policy.name == "allow-ingress-application-client" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        assert simulation.policiesList.find { it.oldPolicy.name == "allow-ingress-application-web" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.DELETED

        // Verify outEdge to 'web' is added to all nodes
        simulation.added.nodeDiffsMap.each {
            if (it.key != clientAppId) {
                assert it.value.outEdgesMap.containsKey(clientAppId)
            } else {
                assert it.value.policyIdsCount > 0
            }
        }

        // Verify removed contains the toDelete policy
        assert simulation.removed.nodeDiffsMap.each { it.value.policyIdsList.contains(policy1.uid) }

        cleanup:
        "cleanup"
        Services.cleanupNetworkPolicies([policy1, policy2])

        where:
        "Data is:"

        scope                           | orchestratorDepsShouldExist
        "Orchestrator Component:false"  | false
        ""                              | true
    }

    @Tag("NetworkPolicySimulation")
    
    def "Verify NetworkPolicy Simulator allow traffic to an application from all namespaces"() {
        when:
        "generate simulation"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-namespace")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-to-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy1) + orchestrator.generateYaml(policy2)
        )
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def clientAppId = simulation.simulatedGraph.nodesList.find {
            it.deploymentName == CLIENTDEPLOYMENT
        }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() > 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, clientAppId).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, clientAppId, null).size() == 0

        assert simulation.policiesList.find { it.policy.name == "deny-all-namespace" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        assert simulation.policiesList.find { it.policy.name == "allow-ingress-to-application-web" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        // Verify added details for deployments in test namespace
        simulation.added.nodeDiffsMap.each { k, v ->
            def newNode = simulation.simulatedGraph.nodesList.find { it.deploymentId == k }
            if (newNode.entity.deployment.namespace == "qa") {
                assert v.policyIdsCount > 0
                assert v.outEdgesCount == 0
            } else {
                assert v.policyIdsCount == 0
            }
            assert !v.nonIsolatedIngress
            assert !v.nonIsolatedEgress
        }

        // Verify removed details contains only deployments from test namespace, or deployments that have an egress
        // network policy applying to them (since for these we store the outgoing edges explicitly).
        simulation.removed.nodeDiffsMap.each { k, v ->
            def origNode = simulation.simulatedGraph.nodesList.find { it.deploymentId == k }
            if (origNode.entity.deployment.namespace == "qa") {
                assert v.policyIdsCount == 0
                assert v.nonIsolatedIngress
                assert v.nonIsolatedEgress
            } else {
                assert !origNode.nonIsolatedEgress
                assert !v.nonIsolatedEgress
                assert !v.nonIsolatedIngress
            }
        }
     }

    @Tag("NetworkPolicySimulation")
    
    def "Verify yaml requires namespace in metadata"() {
        when:
        "create NetworkPolicy object"
        NetworkPolicy policy = new NetworkPolicy("missing-namespace")

        then:
        "attempt to simulate on the yaml"
        assert NetworkPolicyService.submitNetworkGraphSimulation(orchestrator.generateYaml(policy)) == null
    }

    @Tag("NetworkPolicySimulation")
    
    def "Verify malformed yaml returns error"() {
        when:
        "create NetworkPolicy object"
        NetworkPolicy policy = new NetworkPolicy("missing-namespace")

        then:
        "attempt to simulate on the yaml"
        assert NetworkPolicyService.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy)
                        .replaceAll("\\s", "")) == null
        assert NetworkPolicyService.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy) +
                        "ksdmflka\nlsadkfmasl") == null
        assert NetworkPolicyService.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy)
                        .replace("apiVersion:", "apiVersion=")) == null
    }

    @Unroll
    @Tag("NetworkPolicySimulation")
    
    def "Verify NetworkPolicy Simulator results for #policy.name"() {
        when:
        "Get Base Graph"
        def baseline = NetworkPolicyService.getNetworkPolicyGraph()
        def appId = baseline.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId

        then:
        "verify simulation"
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy))?.simulatedGraph
        assert simulation != null
        assert targets == _ ?
                true :
                NetworkGraphUtil.findEdges(simulation, null, appId).size() == targets
        assert sources == _ ?
                true :
                NetworkGraphUtil.findEdges(simulation, appId, null).size() == sources

        where:
        "Data"

        policy                                                  | sources | targets

        // Test 0:
        // Deny all ingress to app
        // target edges for app should drop to 0
        new NetworkPolicy("deny-all-ingress-to-app")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)      | _       | 0

        // Test 1:
        // Deny all egress from app
        // source edges for app should drop to 0
        new NetworkPolicy("deny-all-egress-from-app")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.EGRESS)       | 0       | _

        // Test 2:
        // Deny all egress/ingress from/to app
        // all sources and target edges should drop to 0
        new NetworkPolicy("deny-all-ingress-egress-app")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.EGRESS)
                .addPolicyType(NetworkPolicyTypes.INGRESS)      | 0       | 0

        // Test 3:
        // Allow ingress only from application
        // Add additional deployment to verify communication
        // target edges should drop to 1
        new NetworkPolicy("ingress-only-from-app")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addIngressPodSelector(["app":WEB2DEPLOYMENT])           | _       | 1

        // Test 4:
        // Allow egress only to application
        // Add additional deployment to verify communication
        // source edges should drop to 1
        new NetworkPolicy("egress-only-to-app")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.EGRESS)
                .addEgressPodSelector(["app":WEB2DEPLOYMENT])            | 1       | _

        // Test 5:
        // Deny all ingress traffic
        // Add 2 deployments to verify communication
        // target edges should drop to 0
        new NetworkPolicy("deny-all-ingress")
                .setNamespace("qa")
                .addPolicyType(NetworkPolicyTypes.INGRESS)      | _       | 0

        // Test 6:
        // Deny all egress traffic
        // Add 2 deployments to verify communication
        // source edges should drop to 0
        new NetworkPolicy("deny-all-egress")
                .setNamespace("qa")
                .addPolicyType(NetworkPolicyTypes.EGRESS)       | 0       | _

        // Test 7:
        // Deny all ingress traffic from outside namespaces
        // Add 2 deployments to verify communication
        // target edges should drop to 2
        new NetworkPolicy("deny-all-namespace-ingress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addIngressPodSelector()                     | _       |
                orchestrator.getAllDeploymentTypesCount(Constants.ORCHESTRATOR_NAMESPACE) - 1

        // Test 8:
        // Deny all egress traffic from outside namespaces
        // Add 2 deployments to verify communication
        // source edges should drop to 2
        new NetworkPolicy("deny-all-namespace-egress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.EGRESS)
                .addEgressPodSelector()                      |
                orchestrator.getAllDeploymentTypesCount(Constants.ORCHESTRATOR_NAMESPACE) - 1       | _
    }

    @Tag("NetworkPolicySimulation")
    // skipping tests using SLACK_MAIN_WEBHOOK on P/Z
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify invalid clusterId passed to notification API"() {
        when:
        "create slack notifier"
        SlackNotifier notifier = new SlackNotifier()
        notifier.createNotifier()

        and:
        "create Netowrk Policy yaml"
        NetworkPolicy policy = new NetworkPolicy("test-yaml")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)

        then:
        "notify against invalid clusterId"
        assert NetworkPolicyService.sendSimulationNotification(
                [notifier.id],
                orchestrator.generateYaml(policy),
                "11111111-bbbb-0000-aaaa-111111111111") == null
        assert NetworkPolicyService.sendSimulationNotification(
                [notifier.id],
                orchestrator.generateYaml(policy),
                null) == null
        assert NetworkPolicyService.sendSimulationNotification(
                [notifier.id],
                orchestrator.generateYaml(policy),
                "") == null

        cleanup:
        "remove notifier"
        notifier.deleteNotifier()
    }

    @Tag("NetworkPolicySimulation")
    
    def "Verify invalid notifierId passed to notification API"() {
        when:
        "create Netowrk Policy yaml"
        NetworkPolicy policy = new NetworkPolicy("test-yaml")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)

        then:
        "notify against invalid clusterId"
        assert NetworkPolicyService.sendSimulationNotification(
                ["11111111-bbbb-0000-aaaa-111111111111"],
                orchestrator.generateYaml(policy)) == null
        assert NetworkPolicyService.sendSimulationNotification(
                null,
                orchestrator.generateYaml(policy)) == null
        assert NetworkPolicyService.sendSimulationNotification(
                [""],
                orchestrator.generateYaml(policy)) == null
    }
}
