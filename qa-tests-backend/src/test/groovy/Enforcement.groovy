import groups.BAT
import groups.Integration
import groups.PolicyEnforcement
import objects.DaemonSet
import objects.Deployment
import org.junit.experimental.categories.Category
import spock.lang.Unroll
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.api.v1.PolicyServiceOuterClass.EnforcementAction

class Enforcement extends BaseSpecification {
    private final static String CONTAINER_PORT_22_POLICY = "Secure Shell (ssh) Port Exposed"
    private final static String APT_GET_POLICY = "Ubuntu Package Manager Execution"
    private final static String LATEST_TAG = "Latest tag"

    @Category([BAT, Integration, PolicyEnforcement])
    def "Test Kill Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for Kill Pod enforcement

        given:
        "Add kill enforcement to an existing runtime policy"
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
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                APT_GET_POLICY,
                30
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolation(violations.get(0).id)

        then:
        "check pod was killed"
        def startTime = System.currentTimeMillis()
        assert d.pods.collect {
            it -> println "checking if ${it.name} was killed"
            orchestrator.wasContainerKilled(it.name)
        }.find { it == true }
        assert alert.enforcement.action == EnforcementAction.KILL_POD_ENFORCEMENT
        println "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert Services.getAlertEnforcementCount("kill-enforcement-int", APT_GET_POLICY) > 0

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(APT_GET_POLICY, startEnforcements)
        orchestrator.deleteDeployment(d)
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
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolation(violations.get(0).id)

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
        assert Services.getAlertEnforcementCount(
                "scale-down-enforcement-int",
                CONTAINER_PORT_22_POLICY) == 1

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(CONTAINER_PORT_22_POLICY, startEnforcements)
        orchestrator.deleteDeployment(d)
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
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolation(violations.get(0).id)

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
        assert orchestrator.getDeploymentUnavailableReplicaCount(d) >=
                orchestrator.getDeploymentReplicaCount(d)
        assert alert.enforcement.action == EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT
        assert Services.getAlertEnforcementCount(
                "node-constraint-enforcement-int",
                CONTAINER_PORT_22_POLICY) == 1

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(CONTAINER_PORT_22_POLICY, startEnforcements)
        orchestrator.deleteDeployment(d)
    }

    @Category([BAT, Integration, PolicyEnforcement])
    def "Test Fail Build Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for fail build enforcement

        given:
        "Apply policy at Build time"
        def startlifeCycle = Services.updatePolicyLifecycleStage(
                LATEST_TAG,
                [PolicyServiceOuterClass.LifecycleStage.BUILD,]
        )
        "Add node constraint enforcement to an existing policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                LATEST_TAG,
                [EnforcementAction.FAIL_BUILD_ENFORCEMENT,]
        )

        when:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(
                "apollo-dtr.rox.systems",
                "legacy-apps/struts-app",
                "latest"
        )

        then:
        "verify violation and enforcement"
        assert scanResults.getAlertsList().findAll {
            it.getPolicy().name == LATEST_TAG &&
            it.getPolicy().getEnforcementActionsList().find {
                it.getNumber() == EnforcementAction.FAIL_BUILD_ENFORCEMENT_VALUE
            }
        }.size() == 1

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(LATEST_TAG, startEnforcements)
        Services.updatePolicyLifecycleStage(LATEST_TAG, startlifeCycle)
    }

    @Category([Integration, PolicyEnforcement])
    def "Test Scale-down and Node Selection Enforcement - Deployment"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for scale-down enforcement

        given:
        "Add scale-down and Node Selection enforcement to an existing policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                CONTAINER_PORT_22_POLICY,
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
                ]
        )

        when:
        "Create Deployment to test scale-down and Node Selection enforcement"
        Deployment d = new Deployment()
                .setName("scale-node-deployment-enforcement-int")
                .setImage("nginx")
                .addPort(22)
                .addLabel("app", "scale-node-deployment-enforcement-int")
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(d)

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolation(violations.get(0).id)

        then:
        "check deployment was scaled-down to 0 replicas and node selection was not applied"
        def replicaCount = 1
        def startTime = System.currentTimeMillis()
        while (replicaCount > 0 && (System.currentTimeMillis() - startTime) < 60000) {
            replicaCount = orchestrator.getDeploymentReplicaCount(d)
            sleep 1000
        }
        assert replicaCount == 0
        println "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert alert.enforcement.action == EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT
        //Node Constraint should have been ignored
        assert orchestrator.getDeploymentNodeSelectors(d) == null
        assert orchestrator.getDeploymentUnavailableReplicaCount(d) !=
                orchestrator.getDeploymentReplicaCount(d)
        assert Services.getAlertEnforcementCount(
                "scale-node-deployment-enforcement-int",
                CONTAINER_PORT_22_POLICY) == 1

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(CONTAINER_PORT_22_POLICY, startEnforcements)
        orchestrator.deleteDeployment(d)
    }

    @Category([Integration, PolicyEnforcement])
    def "Test Scale-down and Node Selection Enforcement - DaemonSet"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for scale-down enforcement

        given:
        "Add scale-down and Node Selection enforcement to an existing policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                CONTAINER_PORT_22_POLICY,
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
                ]
        )

        when:
        "Create DaemonSet to test scale-down and Node Selection enforcement"
        DaemonSet d = new DaemonSet()
                .setName("scale-node-daemonset-enforcement-int")
                .setImage("nginx")
                .addPort(22)
                .addLabel("app", "scale-node-daemonset-enforcement-int")
                .setSkipReplicaWait(true)
                .create()

        and:
        "get violation details"
        List<AlertServiceOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertServiceOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertServiceOuterClass.Alert alert = Services.getViolation(violations.get(0).id)

        then:
        "check deployment set with unsatisfiable node constraint, and unavailable nodes = desired nodes"
        def nodeSelectors = null
        def startTime = System.currentTimeMillis()
        while (nodeSelectors == null && (System.currentTimeMillis() - startTime) < 60000) {
            nodeSelectors = orchestrator.getDaemonSetNodeSelectors(d)
            sleep 1000
        }
        assert nodeSelectors != null
        println "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert orchestrator.getDaemonSetUnavailableReplicaCount(d) ==
                orchestrator.getDaemonSetReplicaCount(d)
        assert alert.enforcement.action == EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT
        assert orchestrator.getDaemonSetReplicaCount(d) == 0
        assert Services.getAlertEnforcementCount(
                "scale-node-daemonset-enforcement-int",
                CONTAINER_PORT_22_POLICY) == 1

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(CONTAINER_PORT_22_POLICY, startEnforcements)
        d.delete()
    }

    @Unroll
    @Category([PolicyEnforcement])
    def "Verify Policy Lifecycle combinations: #lifecycles:#policy"() {
        when:
        "attempt to update lifecycle stage for policy"
        def result = Services.updatePolicyLifecycleStage(policy, lifecycles)

        then:
        "verify if update was allowed"
        if (allowed) {
            assert result != []
        } else {
            assert result == []
        }

        cleanup:
        "revert policy lifecycle"
        if (result != []) {
            Services.updatePolicyLifecycleStage(policy, result)
        }

        where:
        "Data inputs:"

        lifecycles                                        | policy         | allowed

        [PolicyServiceOuterClass.LifecycleStage.BUILD,]   | LATEST_TAG     | true

        [PolicyServiceOuterClass.LifecycleStage.DEPLOY,]  | LATEST_TAG     | true

        [PolicyServiceOuterClass.LifecycleStage.BUILD,
         PolicyServiceOuterClass.LifecycleStage.DEPLOY,]  | LATEST_TAG     | true

        [PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | APT_GET_POLICY | true

        [PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [PolicyServiceOuterClass.LifecycleStage.BUILD,
         PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [PolicyServiceOuterClass.LifecycleStage.BUILD,
         PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | APT_GET_POLICY | false

        [PolicyServiceOuterClass.LifecycleStage.DEPLOY,
         PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [PolicyServiceOuterClass.LifecycleStage.DEPLOY,
         PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | APT_GET_POLICY | false

        [PolicyServiceOuterClass.LifecycleStage.BUILD,
         PolicyServiceOuterClass.LifecycleStage.DEPLOY,
         PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [PolicyServiceOuterClass.LifecycleStage.BUILD,
         PolicyServiceOuterClass.LifecycleStage.DEPLOY,
         PolicyServiceOuterClass.LifecycleStage.RUNTIME,] | APT_GET_POLICY | false
    }

    @Unroll
    @Category([PolicyEnforcement])
    def "Verify Policy Enforcement/Lifecycle combinations: #lifecycles"() {
        when:
        "attempt to update lifecycle stage for policy"
        def originalStages = Services.updatePolicyLifecycleStage(policy, lifecycles)
        assert originalStages != []

        and:
        "apply enforcements to the policy"
        def enforcements = EnforcementAction.values() as List
        enforcements.remove(EnforcementAction.UNSET_ENFORCEMENT)
        enforcements.remove(EnforcementAction.UNRECOGNIZED)
        def result = Services.updatePolicyEnforcement(policy, enforcements)
        assert !result.contains("EXCEPTION")

        then:
        "verify if update was allowed"
        assert Services.getPolicyByName(policy).getEnforcementActionsList().containsAll(validEnforcements) &&
                Services.getPolicyByName(policy).getEnforcementActionsList().size() == validEnforcements.size()

        cleanup:
        "revert policy lifecycle"
        Services.updatePolicyLifecycleStage(policy, originalStages)
        if (!result.contains("EXCEPTION")) {
            Services.updatePolicyEnforcement(policy, result)
        }

        where:
        "Data inputs:"

        lifecycles | validEnforcements | policy

        /*
            all-in-one:
         */
        [PolicyServiceOuterClass.LifecycleStage.BUILD,]                        |
                [EnforcementAction.FAIL_BUILD_ENFORCEMENT]                     |
                LATEST_TAG

        [PolicyServiceOuterClass.LifecycleStage.DEPLOY,]                       |
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT]  |
                LATEST_TAG

        [PolicyServiceOuterClass.LifecycleStage.BUILD,
         PolicyServiceOuterClass.LifecycleStage.DEPLOY]                        |
                [EnforcementAction.FAIL_BUILD_ENFORCEMENT,
                 EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT]  |
                LATEST_TAG

        [PolicyServiceOuterClass.LifecycleStage.RUNTIME,]                      |
                [EnforcementAction.KILL_POD_ENFORCEMENT]                       |
                APT_GET_POLICY
    }
}
