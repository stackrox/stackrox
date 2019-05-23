import static Services.waitForViolation

import org.junit.Assume
import groups.BAT
import groups.Integration
import groups.PolicyEnforcement
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ProcessWhitelistOuterClass
import objects.DaemonSet
import objects.Deployment
import org.apache.commons.lang.StringUtils
import org.junit.experimental.categories.Category
import services.AlertService
import services.CreatePolicyService
import services.ProcessWhitelistService
import spock.lang.Shared
import spock.lang.Unroll
import io.stackrox.proto.storage.AlertOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage

class Enforcement extends BaseSpecification {
    private final static String CONTAINER_PORT_22_POLICY = "Secure Shell (ssh) Port Exposed"
    private final static String APT_GET_POLICY = "Ubuntu Package Manager Execution"
    private final static String LATEST_TAG = "Latest tag"
    private final static String CVSS = "Fixable CVSS >= 7"
    private final static String SCAN_AGE = "30-Day Scan Age"
    private final static String WHITELISTPROCESS_POLICY = "Unauthorized Process Execution"

    @Shared
    private String gcrId

    def setupSpec() {
        gcrId = Services.addGcrRegistryAndScanner()
        assert gcrId != null
    }

    def cleanupSpec() {
        assert Services.deleteGcrRegistryAndScanner(gcrId)
    }

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
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                APT_GET_POLICY,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

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
                .setImage("busybox")
                .addPort(22)
                .addLabel("app", "scale-down-enforcement-int")
                .setCommand(["sleep", "600"])
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

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
    def "Test Scale-down Enforcement - Integration (build,deploy - image tag)"() {
        // This test verifies enforcement by triggering a policy violation on an image
        // based policy that is configured for scale-down enforcement with both BUILD and
        // DEPLOY Lifecycle Stages

        given:
        "custom policy to test image tag with enforcement and lifecycle stages"
        PolicyOuterClass.Policy policy = PolicyOuterClass.Policy.newBuilder()
                .setName("TestImageTagPolicyForEnforcement")
                .setDescription("Test image tag")
                .setRationale("Test image tag")
                .addLifecycleStages(LifecycleStage.BUILD)
                .addLifecycleStages(LifecycleStage.DEPLOY)
                .addCategories("Image Assurance")
                .setDisabled(false)
                .setSeverityValue(2)
                .setFields(PolicyOuterClass.PolicyFields.newBuilder()
                        .setImageName(PolicyOuterClass.ImageNamePolicy.newBuilder()
                                .setTag("testing")
                                .build())
                        .build())
                .build()
        String policyID = CreatePolicyService.createNewPolicy(policy)
        assert policyID != null

        and:
        "add enforcement action"
        Services.updatePolicyEnforcement(
                "TestImageTagPolicyForEnforcement",
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.FAIL_BUILD_ENFORCEMENT]
        )

        when:
        "Create Deployment to test scale-down enforcement"
        Deployment d = new Deployment()
                .setName("scale-down-enforcement-build-deploy-image")
                .setImage("apollo-dtr.rox.systems/qa/enforcement:testing")
                .addPort(22)
                .addLabel("app", "scale-down-enforcement-build-deploy")
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                "TestImageTagPolicyForEnforcement",
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

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
                d.name,
                "TestImageTagPolicyForEnforcement") == 1

        cleanup:
        "restore enforcement state of policy and remove deployment"
        orchestrator.deleteDeployment(d)
        if (policyID) {
            CreatePolicyService.deletePolicy(policyID)
        }
    }

    @Category([BAT, Integration, PolicyEnforcement])
    def "Test Scale-down Enforcement - Integration (build,deploy - cvss)"() {
        // This test verifies enforcement by triggering a policy violation on a CVSS
        // based policy that is configured for scale-down enforcement with both BUILD and
        // DEPLOY Lifecycle Stages

        given:
        "add BUILD and DEPLOY lifecycle stages"
        def startlifeCycle = Services.updatePolicyLifecycleStage(
                CVSS,
                [LifecycleStage.BUILD, LifecycleStage.DEPLOY]
        )

        and:
        "Add scale-down and fail-build enforcement to an existing policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                CVSS,
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.FAIL_BUILD_ENFORCEMENT]
        )

        when:
        "Create Deployment to test scale-down enforcement"
        Deployment d = new Deployment()
                .setName("scale-down-enforcement-build-deploy-cvss")
                .setImage("us.gcr.io/stackrox-ci/nginx:1.11")
                .addPort(22)
                .addLabel("app", "scale-down-enforcement-build-deploy")
                .setSkipReplicaWait(true)
                .setCommand(["sleep", "600"])
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CVSS,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

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
                d.name,
                CVSS) == 1

        cleanup:
        "restore enforcement state of policy and remove deployment"
        Services.updatePolicyEnforcement(CVSS, startEnforcements)
        Services.updatePolicyLifecycleStage(CVSS, startlifeCycle)
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
                .setImage("busybox")
                .addPort(22)
                .addLabel("app", "node-constraint-enforcement-int")
                .setCommand(["sleep", "600"])
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(d)
        assert Services.waitForDeployment(d)

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

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
                [LifecycleStage.BUILD, LifecycleStage.DEPLOY]
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

    @Category([BAT, Integration, PolicyEnforcement])
    def "Test Fail Build Enforcement - Integration (build,deploy)"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for fail build enforcement

        given:
        "Apply policy at Build time"
        def startlifeCycle = Services.updatePolicyLifecycleStage(
                LATEST_TAG,
                [LifecycleStage.BUILD, LifecycleStage.DEPLOY]
        )
        "Add node constraint enforcement to an existing policy"
        def startEnforcements = Services.updatePolicyEnforcement(
                LATEST_TAG,
                [EnforcementAction.FAIL_BUILD_ENFORCEMENT, EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT]
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
                .setImage("busybox")
                .addPort(22)
                .addLabel("app", "scale-node-deployment-enforcement-int")
                .setSkipReplicaWait(true)
                .setCommand(["sleep", "600"])
        orchestrator.createDeployment(d)

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

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
                .setImage("busybox")
                .addPort(22)
                .addLabel("app", "scale-node-daemonset-enforcement-int")
                .setSkipReplicaWait(true)
                .setCommand(["sleep", "600"])
                .create()

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                CONTAINER_PORT_22_POLICY,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

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

        [LifecycleStage.BUILD,]   | SCAN_AGE     | true

        [LifecycleStage.DEPLOY,]  | LATEST_TAG     | true

        [LifecycleStage.BUILD,
         LifecycleStage.DEPLOY,]  | LATEST_TAG     | true

        [LifecycleStage.RUNTIME,] | APT_GET_POLICY | true

        [LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [LifecycleStage.BUILD,
         LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [LifecycleStage.BUILD,
         LifecycleStage.RUNTIME,] | APT_GET_POLICY | false

        [LifecycleStage.DEPLOY,
         LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [LifecycleStage.DEPLOY,
         LifecycleStage.RUNTIME,] | APT_GET_POLICY | false

        [LifecycleStage.BUILD,
         LifecycleStage.DEPLOY,
         LifecycleStage.RUNTIME,] | LATEST_TAG     | false

        [LifecycleStage.BUILD,
         LifecycleStage.DEPLOY,
         LifecycleStage.RUNTIME,] | APT_GET_POLICY | false
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
        [LifecycleStage.BUILD,]                        |
                [EnforcementAction.FAIL_BUILD_ENFORCEMENT]                     |
                SCAN_AGE

        [LifecycleStage.DEPLOY,]                       |
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT]  |
                LATEST_TAG

        [LifecycleStage.BUILD,
         LifecycleStage.DEPLOY]                        |
                [EnforcementAction.FAIL_BUILD_ENFORCEMENT,
                 EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT]  |
                LATEST_TAG

        [LifecycleStage.RUNTIME,]                      |
                [EnforcementAction.KILL_POD_ENFORCEMENT]                       |
                APT_GET_POLICY
    }

    @Category([PolicyEnforcement])
    def "Test Alert and  Kill Pod Enforcement - Whitelist Process"() {
        Assume.assumeTrue(false)
        // This test verifies enforcement of kill pod after triggering a policy violation of
        //  Unauthorized Process Execution
        Deployment wpDeployment = new Deployment()
                .setName("deploymentnginx")
                .setImage("nginx:1.7.9")
                .addPort(22, "TCP")
                .addAnnotation("test", "annotation")
                .setEnv(["CLUSTER_NAME": "main"])
                .addLabel("app", "test")
        orchestrator.createDeployment(wpDeployment)
        given:
        "policy violation to whitelist process policy"
        def result = Services.updatePolicyEnforcement(
                WHITELISTPROCESS_POLICY,
                [EnforcementAction.KILL_POD_ENFORCEMENT,
                ]
        )
        assert !result.contains("EXCEPTION")
        when:
        List<ProcessWhitelistOuterClass.ProcessWhitelist> lockProcessWhitelists = ProcessWhitelistService.
                lockProcessWhitelists(wpDeployment.deploymentUid, wpDeployment.name, true)
        assert (!StringUtils.isEmpty(lockProcessWhitelists.get(0).getElements(0).getElement().processName))
        orchestrator.execInContainer(wpDeployment, "pwd")
        assert waitForViolation(wpDeployment.name, WHITELISTPROCESS_POLICY, 90)
        then:
        "check pod was killed"
        List<AlertOuterClass.ListAlert> violations = AlertService.getViolations(AlertServiceOuterClass.ListAlertsRequest
                .newBuilder().build())
        String alertId = violations.find {
            it.getPolicy().name.equalsIgnoreCase(WHITELISTPROCESS_POLICY) &&
            it.deployment.id.equalsIgnoreCase(wpDeployment.deploymentUid) }.id
        assert (alertId != null)
        AlertOuterClass.Alert alert = AlertService.getViolation(alertId)
        def startTime = System.currentTimeMillis()
        assert wpDeployment.pods.collect {
            it ->
            println "checking if ${it.name} was killed"
            orchestrator.wasContainerKilled(it.name)
        }.find { it == true }
        assert alert.enforcement.action == EnforcementAction.KILL_POD_ENFORCEMENT
        println "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert Services.getAlertEnforcementCount(wpDeployment.name, WHITELISTPROCESS_POLICY) > 0

        cleanup:
        "remove deployment"
        if (wpDeployment != null) {
            orchestrator.deleteDeployment(wpDeployment)
        }
        }
    }
