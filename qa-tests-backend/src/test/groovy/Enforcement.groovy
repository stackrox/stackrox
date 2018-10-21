import groups.BAT
import groups.Integration
import groups.PolicyEnforcement
import objects.Deployment
import org.junit.experimental.categories.Category
import stackrox.generated.AlertServiceOuterClass
import stackrox.generated.PolicyServiceOuterClass.EnforcementAction

class Enforcement extends BaseSpecification {
    private final static String CONTAINER_PORT_22_POLICY = "Container Port 22"
    private final static String APT_GET_POLICY = "apt-get Execution"

    @Category([BAT, Integration, PolicyEnforcement])
    def "Test Kill Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for Kill Pod enforcement

        given:
        "Add killn enforcement to an existing runtime policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                APT_GET_POLICY,
                [EnforcementAction.KILL_POD_ENFORCEMENT,]
        )

        when:
        "Create Deployment to test kill enforcement"
        Deployment d = new Deployment()
                .setName("kill-enforcement-int")
                .setImage("nginx")
                .addLabel("app", "kill-enforcement-int")
                .setCommand(["sh" , "-c" , "while true; do sleep 5; apt-get -y update; done"])
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d.deploymentUid)

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                APT_GET_POLICY,
                90
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolaton(violations.get(0).id)

        then:
        "check pod was killed"
        def startTime = System.currentTimeMillis()
        assert d.pods.collect {
            it -> println "checking if ${it.name} was killed"
            orchestrator.wasContainerKilled(it.name)
        }.find { it == true }
        assert alert.enforcement.action == EnforcementAction.KILL_POD_ENFORCEMENT
        println "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(APT_GET_POLICY, startEnforcements)
        orchestrator.deleteDeployment(d.name)
    }

    @Category([BAT, Integration, PolicyEnforcement])
    def "Test Scale-down Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for scale-down enforcement

        given:
        "Add scale-down enforcement to an existing policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                CONTAINER_PORT_22_POLICY,
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        when:
        "Create Deployment to test scale-down enforcement"
        Deployment d = new Deployment()
                .setName("scale-down-enforcement-int")
                .setImage("nginx")
                .addPort(22)
                .addLabel("app", "scale-down-enforcement-int")
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d.deploymentUid)

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolaton(violations.get(0).id)

        then:
        "check deployment was scaled-down to 0 replicas"
        def replicaCount = 1
        def startTime = System.currentTimeMillis()
        while (replicaCount > 0 && (System.currentTimeMillis() - startTime) < 60000) {
            replicaCount = orchestrator.getDeploymentReplicaCount(d)
            sleep 1000
        }
        assert replicaCount == 0
        println "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert alert.enforcement.action == EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(CONTAINER_PORT_22_POLICY, startEnforcements)
        orchestrator.deleteDeployment(d.name)
    }

    @Category([BAT, Integration, PolicyEnforcement])
    def "Test Node Constraint Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for node constraint enforcement

        given:
        "Add node constraint enforcement to an existing policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                CONTAINER_PORT_22_POLICY,
                [EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,]
        )

        when:
        "Create Deployment to test node constraint enforcement"
        Deployment d = new Deployment()
                .setName("node-constraint-enforcement-int")
                .setImage("nginx")
                .addPort(22)
                .addLabel("app", "node-constraint-enforcement-int")
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d.deploymentUid)

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolaton(violations.get(0).id)

        then:
        "check deployment set with unsatisfiable node constraint, and unavailable nodes = desired nodes"
        def nodeSelectors = null
        def startTime = System.currentTimeMillis()
        while (nodeSelectors == null && (System.currentTimeMillis() - startTime) < 60000) {
            nodeSelectors = orchestrator.getDeploymentNodeSelectors(d)
            sleep 1000
        }
        assert nodeSelectors != null
        println "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert orchestrator.getDeploymentUnavailableReplicaCount(d) ==
                orchestrator.getDeploymentReplicaCount(d)
        assert alert.enforcement.action == EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(CONTAINER_PORT_22_POLICY, startEnforcements)
        orchestrator.deleteDeployment(d.name)
    }
}
