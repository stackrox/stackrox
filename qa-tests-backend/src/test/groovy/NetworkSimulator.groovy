import groups.BAT
import groups.NetworkPolicySimulation
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import org.junit.experimental.categories.Category
import spock.lang.Unroll
import stackrox.generated.NotifierServiceOuterClass

class NetworkSimulator extends BaseSpecification {
    @Category([NetworkPolicySimulation, BAT])
    def "Verify NetworkPolicy Simulator replace existing network policy"() {
        when:
        "deploy"
        orchestrator.createDeployment(new Deployment()
                .setName("web")
                .setImage("nginx")
                .addPort(80)
                .addLabel("app", "web")
        )

        and:
        "apply network policy"
        NetworkPolicy policy = new NetworkPolicy("deny-all-namespace-ingress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy)
        def baseline = Services.getNetworkGraph()

        and:
        "generate simulation"
        policy.addPolicyType(NetworkPolicyTypes.EGRESS)
        def simulation = Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy))
        assert simulation != null
        def webAppId = simulation.nodesList.find { it.deploymentName == "web" }.id

        then:
        "verify simulation"
        assert simulation.edgesList.findAll { it.target == webAppId }.size() ==
                baseline.edgesList.findAll { it.target == webAppId }.size()
        assert simulation.edgesList.findAll { it.source == webAppId }.size() == 0

        cleanup:
        "cleanup"
        orchestrator.deleteDeployment("web")
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy)
        }
    }

    @Category([NetworkPolicySimulation, BAT])
    def "Verify NetworkPolicy Simulator add to an existing network policy"() {
        when:
        "deploy"
        orchestrator.createDeployment(new Deployment()
                .setName("web")
                .setImage("nginx")
                .addPort(80)
                .addLabel("app", "web")
        )
        orchestrator.createDeployment(new Deployment()
                .setName("client")
                .setImage("nginx")
                .addPort(443)
                .addLabel("app", "client")
        )

        and:
        "apply network policy"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-traffic")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy1)
        def baseline = Services.getNetworkGraph()

        and:
        "generate simulation"
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": "web"])
                .addIngressNamespaceSelector()
        def simulation = Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy2))
        assert simulation != null
        def webAppId = simulation.nodesList.find { it.deploymentName == "web" }.id
        def clientAppId = simulation.nodesList.find { it.deploymentName == "client" }.id

        then:
        "verify simulation"
        assert simulation.edgesList.findAll { it.target == webAppId }.size() > 0
        assert simulation.edgesList.findAll { it.target == clientAppId }.size() ==
                baseline.edgesList.findAll { it.target == clientAppId }.size()
        assert simulation.edgesList.findAll { it.source == webAppId }.size() ==
                baseline.edgesList.findAll { it.source == webAppId }.size()
        assert simulation.edgesList.findAll { it.source == clientAppId }.size() ==
                baseline.edgesList.findAll { it.source == clientAppId }.size()

        cleanup:
        "cleanup"
        orchestrator.deleteDeployment("web")
        orchestrator.deleteDeployment("client")
        if (policyId != null) {
            orchestrator.deleteNetworkPolicy(policy1)
        }
    }

    @Category([NetworkPolicySimulation])
    def "Verify NetworkPolicy Simulator allow traffic to an application from all namespaces"() {
        when:
        "deploy"
        orchestrator.createDeployment(new Deployment()
                .setName("web")
                .setImage("nginx")
                .addPort(80)
                .addLabel("app", "web")
        )
        orchestrator.createDeployment(new Deployment()
                .setName("client")
                .setImage("nginx")
                .addPort(443)
                .addLabel("app", "client")
        )

        and:
        "generate simulation"
        NetworkPolicy policy1 = new NetworkPolicy("deny-all-namespace")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addPolicyType(NetworkPolicyTypes.EGRESS)
        NetworkPolicy policy2 = new NetworkPolicy("allow-ingress-to-application-web")
                .setNamespace("qa")
                .addPodSelector(["app": "web"])
                .addIngressNamespaceSelector()
        def simulation = Services.submitNetworkGraphSimulation(
                orchestrator.generateYaml(policy1) + orchestrator.generateYaml(policy2)
        )
        assert simulation != null
        def webAppId = simulation.nodesList.find { it.deploymentName == "web" }.id
        def clientAppId = simulation.nodesList.find { it.deploymentName == "client" }.id

        then:
        "verify simulation"
        assert simulation.edgesList.findAll { it.target == webAppId }.size() > 0
        assert simulation.edgesList.findAll { it.target == clientAppId }.size() == 0
        assert simulation.edgesList.findAll { it.source == webAppId }.size() == 0
        assert simulation.edgesList.findAll { it.source == clientAppId }.size() == 0

        cleanup:
        "cleanup"
        orchestrator.deleteDeployment("web")
        orchestrator.deleteDeployment("client")
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
        "deploy"
        orchestrator.createDeployment(new Deployment()
                .setName("web")
                .setImage("nginx")
                .addPort(80)
                .addLabel("app", "web")
        )
        for (Deployment extra : additionalDeployments) {
            orchestrator.createDeployment(extra)
        }
        def baseline = Services.getNetworkGraph()
        def appId = baseline.nodesList.find { it.deploymentName == "web" }.id

        then:
        "verify simulation"
        def simulation = Services.submitNetworkGraphSimulation(orchestrator.generateYaml(policy))
        assert simulation != null
        assert targets == _ ?
                true :
                simulation.edgesList.findAll { it.target == appId }.size() == targets
        assert sources == _ ?
                true :
                simulation.edgesList.findAll { it.source == appId }.size() == sources

        cleanup:
        "cleanup"
        orchestrator.deleteDeployment("web")
        for (Deployment extra : additionalDeployments) {
            orchestrator.deleteDeployment(extra.name)
        }

        where:
        "Data"

        policy                                                  | sources | targets |
                additionalDeployments

        // Test 0:
        // Deny all ingress to app
        // target edges for app should drop to 0
        new NetworkPolicy("deny-all-ingress-to-app")
                .setNamespace("qa")
                .addPodSelector(["app":"web"])
                .addPolicyType(NetworkPolicyTypes.INGRESS)      | _       | 0       |
                []

        // Test 1:
        // Deny all egress from app
        // source edges for app should drop to 0
        new NetworkPolicy("deny-all-egress-from-app")
                .setNamespace("qa")
                .addPodSelector(["app":"web"])
                .addPolicyType(NetworkPolicyTypes.EGRESS)       | 0       | _       |
                []

        // Test 2:
        // Deny all egress/ingress from/to app
        // all sources and target edges should drop to 0
        new NetworkPolicy("deny-all-ingress-egress-app")
                .setNamespace("qa")
                .addPodSelector(["app":"web"])
                .addPolicyType(NetworkPolicyTypes.EGRESS)
                .addPolicyType(NetworkPolicyTypes.INGRESS)      | 0       | 0       |
                []

        // Test 3:
        // Allow ingress only from application
        // Add additional deployment to verify communication
        // target edges should drop to 1
        new NetworkPolicy("ingress-only-from-app")
                .setNamespace("qa")
                .addPodSelector(["app":"web"])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addIngressPodSelector(["app":"web"])           | _       | 1       |
                [new Deployment()
                         .setName("client")
                         .setImage("nginx")
                         .addLabel("app", "web"),]

        // Test 4:
        // Allow egress only to application
        // Add additional deployment to verify communication
        // source edges should drop to 1
        new NetworkPolicy("egress-only-to-app")
                .setNamespace("qa")
                .addPodSelector(["app":"web"])
                .addPolicyType(NetworkPolicyTypes.EGRESS)
                .addEgressPodSelector(["app":"web"])            | 1       | _       |
                [new Deployment()
                         .setName("client")
                         .setImage("nginx")
                         .addLabel("app", "web"),]

        // Test 5:
        // Deny all ingress traffic
        // Add 2 deployments to verify communication
        // target edges should drop to 0
        new NetworkPolicy("deny-all-ingress")
                .setNamespace("qa")
                .addPolicyType(NetworkPolicyTypes.INGRESS)      | _       | 0       |
                [new Deployment()
                         .setName("client1")
                         .setImage("nginx")
                         .addLabel("app", "web"),
                 new Deployment()
                         .setName("client2")
                         .setImage("nginx")
                         .addLabel("app", "client"),]

        // Test 6:
        // Deny all egress traffic
        // Add 2 deployments to verify communication
        // source edges should drop to 0
        new NetworkPolicy("deny-all-namespace-egress")
                .setNamespace("qa")
                .addPolicyType(NetworkPolicyTypes.EGRESS)       | 0       | _       |
                [new Deployment()
                         .setName("client1")
                         .setImage("nginx")
                         .addLabel("app", "web"),
                 new Deployment()
                         .setName("client2")
                         .setImage("nginx")
                         .addLabel("app", "client"),]

        // Test 7:
        // Deny all ingress traffic from outside namespaces
        // Add 2 deployments to verify communication
        // target edges should drop to 2
        new NetworkPolicy("deny-all-namespace-ingress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
                .addIngressPodSelector()                     | _       | 2       |
                [new Deployment()
                         .setName("client1")
                         .setImage("nginx")
                         .addLabel("app", "web"),
                 new Deployment()
                         .setName("client2")
                         .setImage("nginx")
                         .addLabel("app", "client"),]

        // Test 8:
        // Deny all egress traffic from outside namespaces
        // Add 2 deployments to verify communication
        // source edges should drop to 2
        new NetworkPolicy("deny-all-namespace-egress")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.EGRESS)
                .addEgressPodSelector()                      | 2       | _       |
                [new Deployment()
                         .setName("client1")
                         .setImage("nginx")
                         .addLabel("app", "web"),
                 new Deployment()
                         .setName("client2")
                         .setImage("nginx")
                         .addLabel("app", "client"),]
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
                .addPodSelector(["app":"web"])
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
                .addPodSelector(["app":"web"])
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
                .addPodSelector(["app":"web"])
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
