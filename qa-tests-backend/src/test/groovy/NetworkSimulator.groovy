import groups.BAT
import groups.NetworkPolicySimulation
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import org.junit.experimental.categories.Category
import services.NetworkGraphService
import services.NetworkPolicyService
import spock.lang.Unroll
import util.NetworkGraphUtil
import io.stackrox.proto.api.v1.NetworkPolicyServiceOuterClass
import io.stackrox.proto.storage.NotifierOuterClass

class NetworkSimulator extends BaseSpecification {

    // Deployment names
    static final private String WEBDEPLOYMENT = "web"
    static final private String WEB2DEPLOYMENT = "alt-web"
    static final private String CLIENTDEPLOYMENT = "client"
    static final private String CLIENT2DEPLOYMENT = "alt-client"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName(WEBDEPLOYMENT)
                    .setImage("nginx")
                    .addPort(80)
                    .addLabel("app", WEBDEPLOYMENT),
            new Deployment()
                    .setName(WEB2DEPLOYMENT)
                    .setImage("nginx")
                    .addLabel("app", WEB2DEPLOYMENT),
            new Deployment()
                    .setName(CLIENTDEPLOYMENT)
                    .setImage("nginx")
                    .addPort(443)
                    .addLabel("app", CLIENTDEPLOYMENT),
            new Deployment()
                    .setName(CLIENT2DEPLOYMENT)
                    .setImage("nginx")
                    .addLabel("app", CLIENT2DEPLOYMENT),
    ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Category([NetworkPolicySimulation, BAT])
    def "Verify NetworkPolicy Simulator replace existing network policy"() {
        when:
        "apply network policy"
        NetworkPolicy policy = new NetworkPolicy("deny-all-namespace-ingress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy)
        assert NetworkPolicyService.waitForNetworkPolicy(policyId)
        def baseline = NetworkGraphService.getNetworkGraph()

        and:
        "generate simulation"
        policy.addPolicyType(NetworkPolicyTypes.EGRESS)
        def policyYAML = orchestrator.generateYaml(policy)
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(policyYAML)
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() ==
                NetworkGraphUtil.findEdges(baseline, null, webAppId).size()
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert simulation.policiesList.find { it.policy.name == "deny-all-namespace-ingress" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.MODIFIED

        // Verify no change to added nodes
        assert simulation.added.nodeDiffsCount == 0

        // Verify all edges from nodes inside of 'qa' to nodes outside of 'qa' that are ingress isolated are removed
        def nonQANodes = simulation.simulatedGraph.nodesList.findAll {
            it.namespace != "qa" && !it.nonIsolatedIngress
        }.size()
        simulation.removed.nodeDiffsMap.each {
            def node = simulation.simulatedGraph.nodesList.get(it.key)

            assert node.namespace == "qa"
            assert it.value.policyIdsCount == 0
            assert it.value.outEdgesMap.size() == nonQANodes
            assert it.value.getNonIsolatedEgress()
        }

        cleanup:
        "cleanup"
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy)
        }
    }

    @Category([NetworkPolicySimulation, BAT])
    def "Verify NetworkPolicy Simulator add to an existing network policy"() {
        when:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy1)
        assert NetworkPolicyService.waitForNetworkPolicy(policyId)
        def baseline = NetworkGraphService.getNetworkGraph()

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(orchestrator.generateYaml(policy2))
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def webAppIndex = simulation.simulatedGraph.nodesList.indexOf(
                simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }
        )
        def clientAppId = simulation.simulatedGraph.nodesList.find {
            it.deploymentName == CLIENTDEPLOYMENT
        }.deploymentId

        then:
        "verify simulation"
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
            if (it.key != webAppIndex) {
                assert it.value.outEdgesMap.containsKey(webAppIndex)
            } else {
                assert it.value.policyIdsCount > 0
            }
        }

        // Verify removed is not changed
        assert simulation.removed.nodeDiffsMap.size() == 0

        cleanup:
        "cleanup"
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy1)
        }
    }

    @Category([NetworkPolicySimulation, BAT])
    def "Verify NetworkPolicy Simulator with query - multiple policy simulation"() {
        when:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy1)
        assert NetworkPolicyService.waitForNetworkPolicy(policyId)

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
                "Deployment:\"web\",\"client\"+Namespace:qa")
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def webAppIndex = simulation.simulatedGraph.nodesList.indexOf(
                simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }
        )
        def clientAppId = simulation.simulatedGraph.nodesList.find {
            it.deploymentName == CLIENTDEPLOYMENT
        }.deploymentId
        def clientAppIndex = simulation.simulatedGraph.nodesList.indexOf(
                simulation.simulatedGraph.nodesList.find { it.deploymentName == CLIENTDEPLOYMENT }
        )

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() == 1
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, clientAppId).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, clientAppId, null).size() == 1
        assert simulation.simulatedGraph.nodesList.size() == 2

        assert simulation.policiesList.find { it.policy.name == "deny-all-traffic" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED

        assert simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        assert simulation.policiesList.find { it.policy.name == "allow-egress-application-client" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        // Verify outEdge to 'web' is added to 'client' node only
        simulation.added.nodeDiffsMap.each {
            if (it.key == clientAppIndex) {
                assert it.value.outEdgesMap.containsKey(webAppIndex)
                assert it.value.outEdgesCount == 1
            } else {
                assert it.value.policyIdsCount > 0
                assert !it.value.outEdgesMap.containsKey(webAppIndex)
            }
        }

        cleanup:
        "cleanup"
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy1)
        }
    }

    @Category([NetworkPolicySimulation, BAT])
    def "Verify NetworkPolicy Simulator with query - single policy simulation"() {
        when:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy1)
        assert NetworkPolicyService.waitForNetworkPolicy(policyId)

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = NetworkPolicyService.submitNetworkGraphSimulation(orchestrator.generateYaml(policy2),
                "Deployment:\"web\",\"central\"+Namespace:qa,stackrox")
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def webAppIndex = simulation.simulatedGraph.nodesList.indexOf(
                simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }
        )
        def centralAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == "central" }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() == 1
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, centralAppId).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, centralAppId, null).size() == 1
        assert simulation.simulatedGraph.nodesList.size() == 2

        assert simulation.policiesList.find { it.policy.name == "deny-all-traffic" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED

        assert simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }?.status ==
                NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED

        // Verify outEdge to 'web' is added to all nodes
        simulation.added.nodeDiffsMap.each {
            if (it.key != webAppIndex) {
                assert it.value.outEdgesMap.containsKey(webAppIndex)
            } else {
                assert it.value.policyIdsCount > 0
            }
        }

        // Verify removed is not changed
        assert simulation.removed.nodeDiffsMap.size() == 0

        cleanup:
        "cleanup"
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy1)
        }
    }

    @Category([NetworkPolicySimulation])
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
            if (simulation.simulatedGraph.nodesList.get(k).entity.deployment.namespace == "qa") {
                assert v.policyIdsCount > 0
                assert v.outEdgesCount == 0
                assert !v.nonIsolatedIngress
                assert !v.nonIsolatedEgress
            }
        }

        // Verify removed details contains only deployments from test namespace
        simulation.removed.nodeDiffsMap.each { k, v ->
            assert simulation.simulatedGraph.nodesList.get(k).entity.deployment.namespace == "qa"
            assert v.policyIdsCount == 0
            assert v.nonIsolatedIngress
            assert v.nonIsolatedEgress
        }
     }

    @Category([NetworkPolicySimulation])
    def "Verify yaml requires namespace in metadata"() {
        when:
        "create NetworkPolicy object"
        NetworkPolicy policy = new NetworkPolicy("missing-namespace")

        then:
        "attempt to simulate on the yaml"
        assert NetworkPolicyService.submitNetworkGraphSimulation(orchestrator.generateYaml(policy)) == null
    }

    @Category([NetworkPolicySimulation])
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
    @Category([NetworkPolicySimulation])
    def "Verify NetworkPolicy Simulator results"() {
        when:
        "Get Base Graph"
        def baseline = NetworkGraphService.getNetworkGraph()
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
        new NetworkPolicy("deny-all-namespace-egress")
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
                .addIngressPodSelector()                     | _       | DEPLOYMENTS.size() - 1

        // Test 8:
        // Deny all egress traffic from outside namespaces
        // Add 2 deployments to verify communication
        // source edges should drop to 2
        new NetworkPolicy("deny-all-namespace-egress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.EGRESS)
                .addEgressPodSelector()                      | DEPLOYMENTS.size() - 1       | _
    }

    @Unroll
    @Category([NetworkPolicySimulation])
    def "Verify Network Simulator Notifications"() {
        when:
        "create notifier"
        def notifiers = []
        for (String notifierType : notifierTypes) {
            NotifierOuterClass.Notifier notifier
            switch (notifierType) {
                case "SLACK":
                    notifier = Services.addSlackNotifier("Slack Test")
                    break

                case "JIRA":
                    notifier = Services.addJiraNotifier("Jira Test")
                    break

                case "EMAIL":
                    notifier = Services.addEmailNotifier("Email Test")
                    break
            }
            notifiers.add(notifier)
        }
        assert notifiers.size() > 0

        and:
        "generate a network policy yaml"
        NetworkPolicy policy = new NetworkPolicy("test-yaml")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)

        then:
        "send simulation notification"
        NetworkPolicyService.sendSimulationNotification(
                notifiers*.id,
                orchestrator.generateYaml(policy)
        )

        cleanup:
        "delete notifiers"
        for (NotifierOuterClass.Notifier notifier : notifiers) {
            if (notifier != null) {
                Services.deleteNotifier(notifier.id)
            }
        }

        where:
        "notifier types"

        notifierTypes     | _
        ["SLACK"]         | _
        ["EMAIL"]         | _
        ["JIRA"]          | _
        ["JIRA", "EMAIL"] | _
    }

    @Category([NetworkPolicySimulation])
    def "Verify invalid clusterId passed to notification API"() {
        when:
        "create slack notifier"
        NotifierOuterClass.Notifier notifier = Services.addSlackNotifier("Slack Test")

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
        if (notifier != null) {
            Services.deleteNotifier(notifier.id)
        }
    }

    @Category([NetworkPolicySimulation])
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
