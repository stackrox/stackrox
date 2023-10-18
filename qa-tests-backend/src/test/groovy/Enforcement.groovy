import static Services.waitForViolation

import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.storage.AlertOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.ProcessBaselineOuterClass
import io.stackrox.proto.storage.ScopeOuterClass

import objects.DaemonSet
import objects.Deployment
import services.AlertService
import services.ClusterService
import services.PolicyService
import services.ProcessBaselineService
import util.Timer

import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll

@Tag("PZ")
class Enforcement extends BaseSpecification {

    // Test labels - each test has its own unique label space. This is also used to name
    // each tests policy and deployment.
    private final static String KILL_ENFORCEMENT = "kill-enforcement-only"
    private final static String SCALE_DOWN_ENFORCEMENT = "scale-down-enforcement-only"
    private final static String SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE = "scale-down-enforcement-build-deploy-image"
    private final static String SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_SEVERITY =
            "scale-down-enforcement-build-deploy-severity"
    private final static String NODE_CONSTRAINT_ENFORCEMENT = "node-constraint-enforcement"
    private final static String FAIL_BUILD_ENFORCEMENT = "fail-build-enforcement-only"
    private final static String FAIL_BUILD_ENFORCEMENT_WITH_SCALE_TO_ZERO = "fail-build-enforcement-with-scale-to-zero"
    private final static String SCALE_DOWN_AND_NODE_CONSTRAINT = "scale-down-and-node-constraint-deployment"
    private final static String SCALE_DOWN_AND_NODE_CONSTRAINT_FOR_DS = "scale-down-and-node-constraint-daemonset"
    private final static String ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS =
            "alert-and-kill-enforcement-baseline-process"
    private final static String NO_ENFORCEMENT_ON_UPDATE = "no-enforcement-on-update"
    private final static String NO_ENFORCEMENT_WITH_BYPASS_ANNOTATION = "no-enforcement-with-bypass-annotation"

    // Test policies - per test specific copies of well known builtin policies with a new name,
    // limited by app label and with initial enforcement actions.
    private final static Map<String, Closure> POLICIES = [
            (KILL_ENFORCEMENT)                         : {
                duplicatePolicyForTest(
                        APT_GET_POLICY,
                        KILL_ENFORCEMENT,
                        [EnforcementAction.KILL_POD_ENFORCEMENT,])
            },
            (SCALE_DOWN_ENFORCEMENT)                   : {
                duplicatePolicyForTest(
                        CONTAINER_PORT_22_POLICY,
                        SCALE_DOWN_ENFORCEMENT,
                        [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,])
            },
            (SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE): {
                PolicyOuterClass.Policy policy = PolicyOuterClass.Policy.newBuilder()
                        .setName(SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE)
                        .setDescription("Test image tag")
                        .setRationale("Test image tag")
                        .addLifecycleStages(LifecycleStage.BUILD)
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Image Assurance")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addAllEnforcementActions([EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                                                   EnforcementAction.FAIL_BUILD_ENFORCEMENT]
                        )
                        .addScope(
                                ScopeOuterClass.Scope.newBuilder()
                                        .setLabel(ScopeOuterClass.Scope.Label.newBuilder()
                                        .setKey("app").setValue(SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE))
                        )
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Tag")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("enforcement")
                                                        .build()).build()
                                ).build()
                        ).build()
                PolicyService.createNewPolicy(policy)
            },
            (SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_SEVERITY) : {
                duplicatePolicyForTest(
                        SEVERITY,
                        SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_SEVERITY,
                        [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT, EnforcementAction.FAIL_BUILD_ENFORCEMENT],
                        [LifecycleStage.BUILD, LifecycleStage.DEPLOY]
                )
            },
            (NODE_CONSTRAINT_ENFORCEMENT)              : {
                duplicatePolicyForTest(
                        CONTAINER_PORT_22_POLICY,
                        NODE_CONSTRAINT_ENFORCEMENT,
                        [EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,])
            },
            (FAIL_BUILD_ENFORCEMENT) : {
                duplicatePolicyForTest(
                        LATEST_TAG,
                        FAIL_BUILD_ENFORCEMENT,
                        [EnforcementAction.FAIL_BUILD_ENFORCEMENT],
                        [LifecycleStage.BUILD, LifecycleStage.DEPLOY]
                )
            },
            (FAIL_BUILD_ENFORCEMENT_WITH_SCALE_TO_ZERO) : {
                duplicatePolicyForTest(
                        LATEST_TAG,
                        FAIL_BUILD_ENFORCEMENT_WITH_SCALE_TO_ZERO,
                        [EnforcementAction.FAIL_BUILD_ENFORCEMENT, EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT],
                        [LifecycleStage.BUILD, LifecycleStage.DEPLOY]
                )
            },
            (SCALE_DOWN_AND_NODE_CONSTRAINT): {
                duplicatePolicyForTest(
                        CONTAINER_PORT_22_POLICY,
                        SCALE_DOWN_AND_NODE_CONSTRAINT,
                        [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                         EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,],
                )
            },
            (SCALE_DOWN_AND_NODE_CONSTRAINT_FOR_DS): {
                duplicatePolicyForTest(
                        CONTAINER_PORT_22_POLICY,
                        SCALE_DOWN_AND_NODE_CONSTRAINT_FOR_DS,
                        [EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
                         EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,],
                )
            },
            (ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS): {
                duplicatePolicyForTest(
                        BASELINEPROCESS_POLICY,
                        ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS,
                        [EnforcementAction.KILL_POD_ENFORCEMENT],
                )
            },
            (NO_ENFORCEMENT_ON_UPDATE): {
                duplicatePolicyForTest(
                        CONTAINER_PORT_22_POLICY,
                        NO_ENFORCEMENT_ON_UPDATE,
                        [],
                )
            },
            (NO_ENFORCEMENT_WITH_BYPASS_ANNOTATION): {
                duplicatePolicyForTest(
                        CONTAINER_PORT_22_POLICY,
                        NO_ENFORCEMENT_WITH_BYPASS_ANNOTATION,
                        [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT],
                )
            },
    ]

    // Test deployments - the map key will be set as name and "app" label.
    private final static Map<String, Deployment> DEPLOYMENTS = [
            (KILL_ENFORCEMENT):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx")
                            .setCommand(["sh", "-c", "while true; do sleep 5; apt-get -y update; done"])
                            .setSkipReplicaWait(true),
            (SCALE_DOWN_ENFORCEMENT):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                            .addPort(22)
                            .setCommand(["sleep", "600"])
                            .setSkipReplicaWait(true),
            (SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:enforcement")
                            .addPort(22)
                            .setSkipReplicaWait(true),
            (SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_SEVERITY):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-12-1")
                            .addPort(22)
                            .setSkipReplicaWait(true)
                            .setCommand(["sleep", "600"]),
            (NODE_CONSTRAINT_ENFORCEMENT):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                            .addPort(22)
                            .setCommand(["sleep", "600"])
                            .setSkipReplicaWait(true),
            (SCALE_DOWN_AND_NODE_CONSTRAINT):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                            .addPort(22)
                            .setCommand(["sleep", "600"])
                            .setSkipReplicaWait(true),
            (ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS):
                    new Deployment()
                            .setImage(TEST_IMAGE)
                            .addPort(22, "TCP")
                            .addAnnotation("test", "annotation")
                            .setEnv(["CLUSTER_NAME": "main"]),
            (NO_ENFORCEMENT_ON_UPDATE):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                            .addPort(22)
                            .setCommand(["sleep", "600"])
                            .setSkipReplicaWait(true),
            (NO_ENFORCEMENT_WITH_BYPASS_ANNOTATION):
                    new Deployment()
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                            .addPort(22)
                            .setCommand(["sleep", "600"])
                            .addAnnotation("admission.stackrox.io/break-glass", "yay")
                            .setSkipReplicaWait(false),
    ]

    private final static Map<String, DaemonSet> DAEMON_SETS = [
            (SCALE_DOWN_AND_NODE_CONSTRAINT_FOR_DS):
                    new DaemonSet()
                            .setName("dset1")
                            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                            .addPort(22)
                            .setCommand(["sleep", "600"])
                            .setSkipReplicaWait(true) as DaemonSet,
    ]

    // Policies used in this test
    private final static String CONTAINER_PORT_22_POLICY = "Secure Shell (ssh) Port Exposed"
    private final static String APT_GET_POLICY = "Ubuntu Package Manager Execution"
    private final static String LATEST_TAG = "Latest tag"
    private final static String SEVERITY = "Fixable Severity at least Important"
    private final static String SCAN_AGE = "30-Day Scan Age"
    private final static String BASELINEPROCESS_POLICY = "Unauthorized Process Execution"

    @Shared
    private static final Map<String, String> CREATED_POLICIES = [:]

    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = 90

    def setupSpec() {
        POLICIES.each {
            label, create ->
            CREATED_POLICIES[label] = create()
            assert CREATED_POLICIES[label], "${label} policy should have been created"
        }

        log.info "Waiting for policies to propagate..."
        sleep 10000

        orchestrator.batchCreateDeployments(DEPLOYMENTS.collect {
            String label, Deployment d -> d.setName(label).addLabel("app", label)
        })
        DAEMON_SETS.each {
            label, d -> d.setName(label).addLabel("app", label).create()
        }
    }

    def cleanupSpec() {
        CREATED_POLICIES.each {
            unused, policyId -> PolicyService.deletePolicy(policyId)
        }
        DEPLOYMENTS.each {
            label, d -> orchestrator.deleteDeployment(d)
        }
        DAEMON_SETS.each {
            unused, d -> d.delete()
        }
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Kill Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for Kill Pod enforcement

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[KILL_ENFORCEMENT]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                KILL_ENFORCEMENT,
                60
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

        and:
        "check pod was killed"
        def startTime = System.currentTimeMillis()
        assert d.pods.size() > 0
        assert d.pods.collect {
            it -> log.info "checking if ${it.name} was killed"
            orchestrator.wasContainerKilled(it.name)
        }.find { it == true }
        assert alert.enforcement.action == EnforcementAction.KILL_POD_ENFORCEMENT
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert Services.getAlertEnforcementCount(KILL_ENFORCEMENT, KILL_ENFORCEMENT) > 0
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Scale-down Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for scale-down enforcement

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[SCALE_DOWN_ENFORCEMENT]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                SCALE_DOWN_ENFORCEMENT,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

        and:
        "check deployment was scaled-down to 0 replicas"
        def replicaCount = orchestrator.getDeploymentReplicaCount(d)
        def startTime = System.currentTimeMillis()
        while (replicaCount > 0 && (System.currentTimeMillis() - startTime) < 60000) {
            replicaCount = orchestrator.getDeploymentReplicaCount(d)
            sleep 1000
        }
        assert replicaCount == 0
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert alert.enforcement.action == EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT
        assert Services.getAlertEnforcementCount(
                SCALE_DOWN_ENFORCEMENT,
                SCALE_DOWN_ENFORCEMENT) == 1
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Scale-down Enforcement - Integration (build,deploy - image tag)"() {
        // This test verifies enforcement by triggering a policy violation on an image
        // based policy that is configured for scale-down enforcement with both BUILD and
        // DEPLOY Lifecycle Stages

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

        and:
        "check deployment was scaled-down to 0 replicas"
        def replicaCount = orchestrator.getDeploymentReplicaCount(d)
        def startTime = System.currentTimeMillis()
        while (replicaCount > 0 && (System.currentTimeMillis() - startTime) < 60000) {
            replicaCount = orchestrator.getDeploymentReplicaCount(d)
            sleep 1000
        }
        assert replicaCount == 0
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert alert.enforcement.action == EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT
        assert Services.getAlertEnforcementCount(
                d.name,
                SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_IMAGE) == 1
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Scale-down Enforcement - Integration (build,deploy - SEVERITY)"() {
        // This test verifies enforcement by triggering a policy violation on a SEVERITY
        // based policy that is configured for scale-down enforcement with both BUILD and
        // DEPLOY Lifecycle Stages

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_SEVERITY]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_SEVERITY,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

        and:
        "check deployment was scaled-down to 0 replicas"
        def replicaCount = orchestrator.getDeploymentReplicaCount(d)
        def startTime = System.currentTimeMillis()
        while (replicaCount > 0 && (System.currentTimeMillis() - startTime) < 60000) {
            replicaCount = orchestrator.getDeploymentReplicaCount(d)
            sleep 1000
        }
        assert replicaCount == 0
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert alert.enforcement.action == EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT
        assert Services.getAlertEnforcementCount(
                d.name,
                SCALE_DOWN_ENFORCEMENT_BUILD_DEPLOY_SEVERITY) == 1
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Node Constraint Enforcement - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for node constraint enforcement

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[NODE_CONSTRAINT_ENFORCEMENT]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                NODE_CONSTRAINT_ENFORCEMENT,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

        and:
        "check deployment set with unsatisfiable node constraint, and unavailable nodes = desired nodes"
        def nodeSelectors = null
        def startTime = System.currentTimeMillis()
        while (nodeSelectors == null && (System.currentTimeMillis() - startTime) < 60000) {
            nodeSelectors = orchestrator.getDeploymentNodeSelectors(d)
            sleep 1000
        }
        assert nodeSelectors != null
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert orchestrator.getDeploymentUnavailableReplicaCount(d) >=
                orchestrator.getDeploymentReplicaCount(d)
        assert alert.enforcement.action == EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT
        assert Services.getAlertEnforcementCount(
                d.name,
                NODE_CONSTRAINT_ENFORCEMENT) == 1
    }

    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Fail Build Enforcement - #policyName - Integration (build,deploy)"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for fail build enforcement

        given:
        "policy already fabricated"

        and:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(
                "quay.io",
                "rhacs-eng/qa",
                "latest"
        )

        expect:
        "verify violation and enforcement"
        assert scanResults.getAlertsList().findAll {
            it.getPolicy().name == policyName &&
            it.getPolicy().getEnforcementActionsList().find {
                it.getNumber() == EnforcementAction.FAIL_BUILD_ENFORCEMENT_VALUE
            }
        }.size() == 1

        where:
        policyName | _
        FAIL_BUILD_ENFORCEMENT | _
        FAIL_BUILD_ENFORCEMENT_WITH_SCALE_TO_ZERO | _
    }

    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Scale-down and Node Constraint Enforcement - Deployment"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for scale-down enforcement

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[SCALE_DOWN_AND_NODE_CONSTRAINT]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                SCALE_DOWN_AND_NODE_CONSTRAINT,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

        and:
        "check deployment was scaled-down to 0 replicas and node selection was not applied"
        def replicaCount = orchestrator.getDeploymentReplicaCount(d)
        def startTime = System.currentTimeMillis()
        while (replicaCount > 0 && (System.currentTimeMillis() - startTime) < 60000) {
            replicaCount = orchestrator.getDeploymentReplicaCount(d)
            sleep 1000
        }
        assert replicaCount == 0
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert alert.enforcement.action == EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT
        //Node Constraint should have been ignored
        assert !orchestrator.getDeploymentNodeSelectors(d)
        assert orchestrator.getDeploymentUnavailableReplicaCount(d) !=
                orchestrator.getDeploymentReplicaCount(d)
        assert Services.getAlertEnforcementCount(
                d.name,
                SCALE_DOWN_AND_NODE_CONSTRAINT) == 1
    }

    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Scale-down and Node Constraint Enforcement - DaemonSet"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for scale-down enforcement

        given:
        "policy and daemon set already fabricated"
        DaemonSet d = DAEMON_SETS[SCALE_DOWN_AND_NODE_CONSTRAINT_FOR_DS]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                SCALE_DOWN_AND_NODE_CONSTRAINT_FOR_DS,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)

        and:
        "check deployment set with unsatisfiable node constraint, and unavailable nodes = desired nodes"
        def nodeSelectors = null
        def startTime = System.currentTimeMillis()
        while (nodeSelectors == null && (System.currentTimeMillis() - startTime) < 60000) {
            nodeSelectors = orchestrator.getDaemonSetNodeSelectors(d)
            sleep 1000
        }
        assert nodeSelectors != null
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert orchestrator.getDaemonSetUnavailableReplicaCount(d) ==
                orchestrator.getDaemonSetReplicaCount(d)
        assert alert.enforcement.action == EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT
        assert orchestrator.getDaemonSetReplicaCount(d) == 0
        assert Services.getAlertEnforcementCount(
                d.name,
                SCALE_DOWN_AND_NODE_CONSTRAINT_FOR_DS) == 1
    }

    @Unroll
    @Tag("PolicyEnforcement")
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

        lifecycles                | policy         | allowed

        [LifecycleStage.BUILD,]   | SCAN_AGE       | true

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
    @Tag("PolicyEnforcement")
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
        List<EnforcementAction> result = Services.updatePolicyEnforcement(policy, enforcements, false)

        then:
        "verify if update was allowed"
        assert Services.getPolicyByName(policy).getEnforcementActionsList().containsAll(validEnforcements) &&
                Services.getPolicyByName(policy).getEnforcementActionsList().size() == validEnforcements.size()

        cleanup:
        "revert policy lifecycle"
        Services.updatePolicyLifecycleStage(policy, originalStages)
        Services.updatePolicyEnforcement(policy, result, false)

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
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
                 EnforcementAction.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT,
                 EnforcementAction.FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT]  |
                LATEST_TAG

        [LifecycleStage.BUILD,
         LifecycleStage.DEPLOY]                        |
                [EnforcementAction.FAIL_BUILD_ENFORCEMENT,
                 EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,
                 EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
                 EnforcementAction.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT,
                 EnforcementAction.FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT]  |
                LATEST_TAG

        [LifecycleStage.RUNTIME,]                      |
                [EnforcementAction.KILL_POD_ENFORCEMENT,
                 EnforcementAction.FAIL_KUBE_REQUEST_ENFORCEMENT]              |
                APT_GET_POLICY
    }

    @Tag("BAT")
    @Tag("PolicyEnforcement")
    def "Test Alert and Kill Pod Enforcement - Baseline Process"() {
        // This test verifies enforcement of kill pod after triggering a policy violation of
        //  Unauthorized Process Execution
        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS]

        when:
        String clusterId = ClusterService.getClusterId()
        ProcessBaselineOuterClass.ProcessBaseline baseline = ProcessBaselineService.
                getProcessBaseline(clusterId, d)
        assert (baseline != null)
        log.info baseline.toString()
        List<ProcessBaselineOuterClass.ProcessBaseline> lockProcessBaselines = ProcessBaselineService.
                lockProcessBaselines(clusterId, d, "", true)
        assert lockProcessBaselines.size() ==  1
        assert  lockProcessBaselines.get(0).getElementsList().
                find { it.element.processName.equalsIgnoreCase("/usr/sbin/nginx") } != null
        orchestrator.execInContainer(d, "pwd")
        assert waitForViolation(d.name, ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS, WAIT_FOR_VIOLATION_TIMEOUT)

        then:
        "check pod was killed"
        List<AlertOuterClass.ListAlert> violations = AlertService.getViolations(AlertServiceOuterClass.ListAlertsRequest
                .newBuilder().build())
        String alertId = violations.find {
            it.getPolicy().name.equalsIgnoreCase(ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS) &&
            it.deployment.id.equalsIgnoreCase(d.deploymentUid) }?.id
        assert (alertId != null)
        AlertOuterClass.Alert alert = AlertService.getViolation(alertId)
        assert alert != null

        def startTime = System.currentTimeMillis()
        assert d.pods.collect {
            it ->
            log.info "checking if ${it.name} was killed"
            orchestrator.wasContainerKilled(it.name)
        }.find { it == true }
        assert alert.enforcement.action == EnforcementAction.KILL_POD_ENFORCEMENT
        log.info "Enforcement took ${(System.currentTimeMillis() - startTime) / 1000}s"
        assert Services.getAlertEnforcementCount(d.name, ALERT_AND_KILL_ENFORCEMENT_BASELINE_PROCESS) > 0

        cleanup:
        if (alertId != null) {
            AlertService.resolveAlert(alertId, false)
        }
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Enforcement not done on updated - Integration"() {
        // This test verifies enforcement by triggering a policy violation on a policy
        // that is configured for scale-down enforcement, but not applying enforcements because
        // the policy is only violated once the deployment has been updated

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[NO_ENFORCEMENT_ON_UPDATE]

        and:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                NO_ENFORCEMENT_ON_UPDATE,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0

        and:
        "not scaled down"
        assert orchestrator.getDeploymentReplicaCount(d) == 1

        when:
        "Add scale-down enforcement to an existing policy"
        Services.updatePolicyEnforcement(
                NO_ENFORCEMENT_ON_UPDATE,
                [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,],
                false
        )

        and:
        "Update deployment to have 2 replicas to potentially trigger enforcement"
        d.replicas = 2
        orchestrator.updateDeployment(d)

        then:
        "check deployment was NOT scaled-down to 0 replicas"
        // Wait for 10s to ensure that the deployment was not scaled down

        Timer t = new Timer(10, 1)
        log.info "Verifying that enforcement action was not taken"
        while (t.IsValid()) {
            assert orchestrator.getDeploymentReplicaCount(d) != 0
        }
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("PolicyEnforcement")
    def "Test Scale-down Enforcement Ignored due to Bypass Annotation - Integration"() {
        // This test verifies enforcement is skipped by triggering a policy violation on a policy
        // that is configured for scale-down enforcement with a deployment that carries a bypass
        // annotation.

        given:
        "policy and deployment already fabricated"
        Deployment d = DEPLOYMENTS[NO_ENFORCEMENT_WITH_BYPASS_ANNOTATION]

        expect:
        "get violation details"
        List<AlertOuterClass.ListAlert> violations = Services.getViolationsWithTimeout(
                d.name,
                NO_ENFORCEMENT_WITH_BYPASS_ANNOTATION,
                30
        ) as List<AlertOuterClass.ListAlert>
        assert violations != null && violations?.size() > 0
        AlertOuterClass.Alert alert = AlertService.getViolation(violations.get(0).id)
        assert alert != null
        assert alert?.enforcement?.action == EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT

        and:
        "check deployment did not actually get scaled down"
        def t = new Timer(15, 1)
        while (t.IsValid()) {
            def replicaCount = orchestrator.getDeploymentReplicaCount(d)
            assert replicaCount > 0
        }
    }

    static String duplicatePolicyForTest(
            String policyName,
            String appLabel,
            List<EnforcementAction> enforcementActions,
            List<LifecycleStage> stages = []
    ) {
        PolicyOuterClass.Policy policyMeta = Services.getPolicyByName(policyName)

        def builder = PolicyOuterClass.Policy.newBuilder(policyMeta)

        builder.setId("")
        builder.setName(appLabel)

        builder.addScope(
                ScopeOuterClass.Scope.newBuilder().
                        setLabel(ScopeOuterClass.Scope.Label.newBuilder()
                                .setKey("app").setValue(appLabel)))

        builder.clearEnforcementActions()
        if (enforcementActions != null && !enforcementActions.isEmpty()) {
            builder.addAllEnforcementActions(enforcementActions)
        } else {
            builder.addAllEnforcementActions([])
        }
        if (stages != []) {
            builder.clearLifecycleStages()
            builder.addAllLifecycleStages(stages)
        }

        def policyDef = builder.build()

        return PolicyService.createNewPolicy(policyDef)
    }
}
