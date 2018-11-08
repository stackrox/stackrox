import groups.BAT
import groups.NetworkPolicySimulation
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import org.junit.experimental.categories.Category
import spock.lang.Unroll
import stackrox.generated.NotifierServiceOuterClass
import util.NetworkGraphUtil
import v1.NetworkPolicyServiceOuterClass

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
        assert Services.waitForNetworkPolicy(policyId)
        def baseline = Services.getNetworkGraph()

        and:
        "generate simulation"
        policy.addPolicyType(NetworkPolicyTypes.EGRESS)
        def simulation = Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy))
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() ==
                NetworkGraphUtil.findEdges(baseline, null, webAppId).size()
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim =
                simulation.policiesList.find { it.policy.name == "deny-all-namespace-ingress" }
        assert sim.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.MODIFIED
        assert sim.policy.spec.policyTypesList.containsAll(
                [NetworkPolicyServiceOuterClass.NetworkPolicyType.INGRESS_NETWORK_POLICY_TYPE,
                 NetworkPolicyServiceOuterClass.NetworkPolicyType.EGRESS_NETWORK_POLICY_TYPE,])
        assert sim.oldPolicy.spec.policyTypesList.containsAll(
                [NetworkPolicyServiceOuterClass.NetworkPolicyType.INGRESS_NETWORK_POLICY_TYPE,])

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
        assert Services.waitForNetworkPolicy(policyId)
        def baseline = Services.getNetworkGraph()

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy2))
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
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

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim1 =
                simulation.policiesList.find { it.policy.name == "deny-all-traffic" }
        assert sim1.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED
        assert sim1.policy.spec.policyTypesList.containsAll(
                [NetworkPolicyServiceOuterClass.NetworkPolicyType.INGRESS_NETWORK_POLICY_TYPE,
                 NetworkPolicyServiceOuterClass.NetworkPolicyType.EGRESS_NETWORK_POLICY_TYPE,])

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim2 =
                simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }
        assert sim2.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED
        assert sim2.policy.spec.podSelector.matchLabelsMap.containsKey("app")
        assert sim2.policy.spec.podSelector.matchLabelsMap.get("app") == WEBDEPLOYMENT

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
        assert Services.waitForNetworkPolicy(policyId)

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressPodSelector(["app": CLIENTDEPLOYMENT])
        NetworkPolicy policy3 = new NetworkPolicy("allow-egress-application-client")
                .setNamespace("qa")
                .addPodSelector(["app": CLIENTDEPLOYMENT])
                .addEgressPodSelector(["app": WEBDEPLOYMENT])
        def simulation = Services.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy2) + orchestrator.generateYaml(policy3),
                "Deployment:web,client")
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def clientAppId = simulation.simulatedGraph.nodesList.find {
            it.deploymentName == CLIENTDEPLOYMENT
        }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() == 1
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, clientAppId).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, clientAppId, null).size() == 1
        assert simulation.simulatedGraph.nodesList.size() == 2

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim1 =
                simulation.policiesList.find { it.policy.name == "deny-all-traffic" }
        assert sim1.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED
        assert sim1.policy.spec.policyTypesList.containsAll(
                [NetworkPolicyServiceOuterClass.NetworkPolicyType.INGRESS_NETWORK_POLICY_TYPE,
                 NetworkPolicyServiceOuterClass.NetworkPolicyType.EGRESS_NETWORK_POLICY_TYPE,])

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim2 =
                simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }
        assert sim2.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED
        assert sim2.policy.spec.podSelector.matchLabelsMap.containsKey("app") &&
                sim2.policy.spec.podSelector.matchLabelsMap.get("app") == WEBDEPLOYMENT
        assert sim2.policy.spec.ingressList.find {
            it.fromList.find {
                it.podSelector.matchLabelsMap.containsKey("app") &&
                        it.podSelector.matchLabelsMap.get("app") == CLIENTDEPLOYMENT
            }
        }

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim3 =
                simulation.policiesList.find { it.policy.name == "allow-egress-application-client" }
        assert sim3.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED
        assert sim3.policy.spec.podSelector.matchLabelsMap.containsKey("app") &&
                sim3.policy.spec.podSelector.matchLabelsMap.get("app") == CLIENTDEPLOYMENT
        assert sim3.policy.spec.egressList.find {
            it.toList.find {
                it.podSelector.matchLabelsMap.containsKey("app") &&
                        it.podSelector.matchLabelsMap.get("app") == WEBDEPLOYMENT
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
        assert Services.waitForNetworkPolicy(policyId)

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": WEBDEPLOYMENT])
                .addIngressNamespaceSelector()
        def simulation = Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy2),
                "Deployment:web,central")
        assert simulation != null
        def webAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId
        def centralAppId = simulation.simulatedGraph.nodesList.find { it.deploymentName == "central" }.deploymentId

        then:
        "verify simulation"
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, webAppId).size() == 1
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, null, centralAppId).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, webAppId, null).size() == 0
        assert NetworkGraphUtil.findEdges(simulation.simulatedGraph, centralAppId, null).size() == 1
        assert simulation.simulatedGraph.nodesList.size() == 2

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim1 =
                simulation.policiesList.find { it.policy.name == "deny-all-traffic" }
        assert sim1.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.UNCHANGED
        assert sim1.policy.spec.policyTypesList.containsAll(
                [NetworkPolicyServiceOuterClass.NetworkPolicyType.INGRESS_NETWORK_POLICY_TYPE,
                 NetworkPolicyServiceOuterClass.NetworkPolicyType.EGRESS_NETWORK_POLICY_TYPE,])

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim2 =
                simulation.policiesList.find { it.policy.name == "allow-ingress-application-web" }
        assert sim2.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED
        assert sim2.policy.spec.podSelector.matchLabelsMap.containsKey("app")
        assert sim2.policy.spec.podSelector.matchLabelsMap.get("app") == WEBDEPLOYMENT

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
        def simulation = Services.submitNetworkGraphSimulation(
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

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim1 =
                simulation.policiesList.find { it.policy.name == "deny-all-namespace" }
        assert sim1.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED
        assert sim1.policy.spec.policyTypesList.containsAll(
                [NetworkPolicyServiceOuterClass.NetworkPolicyType.INGRESS_NETWORK_POLICY_TYPE,
                 NetworkPolicyServiceOuterClass.NetworkPolicyType.EGRESS_NETWORK_POLICY_TYPE,])

        NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation sim2 =
                simulation.policiesList.find { it.policy.name == "allow-ingress-to-application-web" }
        assert sim2.status == NetworkPolicyServiceOuterClass.NetworkPolicyInSimulation.Status.ADDED
        assert sim2.policy.spec.podSelector.matchLabelsMap.containsKey("app")
        assert sim2.policy.spec.podSelector.matchLabelsMap.get("app") == WEBDEPLOYMENT
     }

    @Category([NetworkPolicySimulation])
    def "Verify yaml requires namespace in metadata"() {
        when:
        "create NetworkPolicy object"
        NetworkPolicy policy = new NetworkPolicy("missing-namespace")

        then:
        "attempt to simulate on the yaml"
        assert Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy)) == null
    }

    @Category([NetworkPolicySimulation])
    def "Verify malformed yaml returns error"() {
        when:
        "create NetworkPolicy object"
        NetworkPolicy policy = new NetworkPolicy("missing-namespace")

        then:
        "attempt to simulate on the yaml"
        assert Services.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy)
                        .replaceAll("\\s", "")) == null
        assert Services.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy) +
                        "ksdmflka\nlsadkfmasl") == null
        assert Services.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy)
                        .replace("apiVersion:", "apiVersion=")) == null
    }

    @Unroll
    @Category([NetworkPolicySimulation])
    def "Verify NetworkPolicy Simulator results"() {
        when:
        "Get Base Graph"
        def baseline = Services.getNetworkGraph()
        def appId = baseline.nodesList.find { it.deploymentName == WEBDEPLOYMENT }.deploymentId

        then:
        "verify simulation"
        def simulation = Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy))?.simulatedGraph
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
        NotifierServiceOuterClass.Notifier notifier
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
        assert notifier != null

        and:
        "generate a network policy yaml"
        NetworkPolicy policy = new NetworkPolicy("test-yaml")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)

        then:
        "send simulation notification"
        Services.sendSimulationNotification(
                notifier.id,
                orchestrator.generateYaml(policy)
        )

        cleanup:
        "delete notifier"
        if (notifier != null) {
            Services.deleteNotifier(notifier.id)
        }

        where:
        "notifier types"

        notifierType | _
        "SLACK"      | _
        "EMAIL"      | _
        "JIRA"       | _
    }

    @Category([NetworkPolicySimulation])
    def "Verify invalid clusterId passed to notification API"() {
        when:
        "create slack notifier"
        NotifierServiceOuterClass.Notifier notifier = Services.addSlackNotifier("Slack Test")

        and:
        "create Netowrk Policy yaml"
        NetworkPolicy policy = new NetworkPolicy("test-yaml")
                .setNamespace("qa")
                .addPodSelector(["app":WEBDEPLOYMENT])
                .addPolicyType(NetworkPolicyTypes.INGRESS)

        then:
        "notify against invalid clusterId"
        assert Services.sendSimulationNotification(
                notifier.id,
                orchestrator.generateYaml(policy),
                "11111111-bbbb-0000-aaaa-111111111111") == null
        assert Services.sendSimulationNotification(
                notifier.id,
                orchestrator.generateYaml(policy),
                null) == null
        assert Services.sendSimulationNotification(
                notifier.id,
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
        assert Services.sendSimulationNotification(
                "11111111-bbbb-0000-aaaa-111111111111",
                orchestrator.generateYaml(policy)) == null
        assert Services.sendSimulationNotification(
                null,
                orchestrator.generateYaml(policy)) == null
        assert Services.sendSimulationNotification(
                "",
                orchestrator.generateYaml(policy)) == null
    }
}
