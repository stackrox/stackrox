import static util.Helpers.withRetry

import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.storage.AlertOuterClass.ListAlert
import io.stackrox.proto.storage.AlertOuterClass.ViolationState
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.Policy

import objects.Deployment
import services.AlertService
import services.ClusterService
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Stepwise
import spock.lang.Tag
import spock.lang.Unroll

@Stepwise
class AttemptedAlertsTest extends BaseSpecification {
    static final private String DEP_PREFIX = "attempted-alerts-dep"
    static final private String[] DEP_NAMES = getDeploymentNames()
    private final static Map<String, Deployment> DEPLOYMENTS = [
            (DEP_NAMES[0]): createDeployment(DEP_NAMES[0], "quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),
            (DEP_NAMES[1]): createDeployment(DEP_NAMES[1], "quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),
            (DEP_NAMES[2]): createDeployment(DEP_NAMES[2], "quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),
            (DEP_NAMES[3]): createDeployment(DEP_NAMES[3], "quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),
            (DEP_NAMES[4]): createDeployment(DEP_NAMES[4], "quay.io/rhacs-eng/qa-multi-arch:nginx-1-14-alpine"),
            (DEP_NAMES[5]): createDeployment(DEP_NAMES[5], "quay.io/rhacs-eng/qa-multi-arch:nginx-1-14-alpine"),
    ]

    static final private String LATEST_TAG_POLICY_NAME = "Latest tag"
    static final private String KUBECTL_EXEC_POLICY_NAME = "Kubernetes Actions: Exec into Pod"
    static final private List<String> POLICY_NAMES = [LATEST_TAG_POLICY_NAME, KUBECTL_EXEC_POLICY_NAME]

    static final private List<Policy> OLD_POLICIES = []

    static final private List<EnforcementAction> NO_ENFORCEMENTS = []
    static final private List<EnforcementAction> DEPLOY_TIME_ENFORCEMENTS =
            [EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
    static final private List<EnforcementAction> RUN_TIME_ENFORCEMENTS =
            [EnforcementAction.FAIL_KUBE_REQUEST_ENFORCEMENT,]

    @Shared
    private AdmissionControllerConfig oldAdmCtrlConfig

    private static getDeploymentNames() {
        String[] names = new String[6]
        for (int i = 0; i < 6; i++) {
            names[i] = new StringBuilder().append(DEP_PREFIX).append("-").append(i+1).toString()
        }
        return names
    }

    private static createDeployment(String name, String image) {
        Deployment deployment = new Deployment()
                .setName(name)
                .setImage(image)
                .addLabel("app", "test")
        return deployment
    }

    def setupSpec() {
        def clusterId = ClusterService.getClusterId()
        assert clusterId

        for (def policyName : POLICY_NAMES) {
            def policy = Services.getPolicyByName(policyName)
            assert policy && policy.getName() == policyName && !policy.getDisabled()
            OLD_POLICIES.add(policy)
        }

        oldAdmCtrlConfig = ClusterService.getCluster().getDynamicConfig().getAdmissionControllerConfig()
    }

    def cleanup() {
        for (def policy : OLD_POLICIES) {
            Services.updatePolicy(policy)
        }
        assert ClusterService.updateAdmissionController(oldAdmCtrlConfig)
    }

    def cleanupSpec() {
        for (def deployment : DEPLOYMENTS.values()) {
            orchestrator.deleteDeployment(deployment)
        }

        for (def policyName : POLICY_NAMES) {
            def alerts = Services.getViolationsWithTimeout(DEP_PREFIX, policyName, 0)
            for (def oldAlert : alerts) {
                AlertService.resolveAlert(oldAlert.getId())
            }
        }
    }

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("PZ")
    def "Verify attempted alerts on deployment create: #desc"() {
        when:
        "Set 'Latest Tag' policy enforcement to #policyEnforcements"
        Services.updatePolicyEnforcement(LATEST_TAG_POLICY_NAME, policyEnforcements, true)
        def policy = Services.getPolicyByName(LATEST_TAG_POLICY_NAME)
        assert policy && policy.getName() == LATEST_TAG_POLICY_NAME
        assert policy.enforcementActionsList == policyEnforcements

        and:
        "Set admission controller settings to enforce on creates to #enforce"
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(enforce)
                .setTimeoutSeconds(3)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Sleep to allow settings update to propagate
        sleep(5000)

        and:
        "Trigger create deployment #deploymentName"
        def created = orchestrator.createDeploymentNoWait(DEPLOYMENTS.get(deploymentName))

        then:
        "Verify deployment create"
        assert created == createShouldSucceed

        and:
        "Verify alerts"
        List<ListAlert> listAlerts = []
        withRetry(3, 3) {
            listAlerts = Services.getViolationsWithTimeout(deploymentName, LATEST_TAG_POLICY_NAME, 60)
            // Expected number of alerts for deployment name relies on the order of data inputs.
            assert listAlerts && listAlerts.size() == numAlerts
            assert listAlerts.get(0).getPolicy().getName() == LATEST_TAG_POLICY_NAME

            // Alerts are sorted in descending order of their violation time, therefore, the following check is
            // applied to most recent violations.
            if (createShouldSucceed) {
                assert listAlerts.get(0).getState() == ViolationState.ACTIVE
            } else {
                assert listAlerts.get(0).getState() == ViolationState.ATTEMPTED
                // Verify admission controller enforcement action is applied.
                assert listAlerts.get(0).getEnforcementAction() == EnforcementAction.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT
            }
        }

        // Verify that the alerts are not merged.
        assert AlertService.getViolation(listAlerts.get(0).getId()).getViolationsList().size() == 1

        where:
        "Data inputs are: "

        enforce | deploymentName | policyEnforcements       | createShouldSucceed | numAlerts |
                desc
        false   | DEP_NAMES[0]   | DEPLOY_TIME_ENFORCEMENTS | true                | 1         |
                "no create enforce; policy enforce"
        true    | DEP_NAMES[1]   | DEPLOY_TIME_ENFORCEMENTS | false               | 1         |
                "create enforce; policy enforce"
        true    | DEP_NAMES[1]   | DEPLOY_TIME_ENFORCEMENTS | false               | 2         |
                "create enforce; policy enforce; 2nd attempt"
        // 1 active and 2 attempted alerts are expected.
        false   | DEP_NAMES[1]   | DEPLOY_TIME_ENFORCEMENTS | true                | 3         |
                "no create enforce; policy enforce; 2nd attempt"
        false   | DEP_NAMES[2]   | NO_ENFORCEMENTS          | true                | 1         |
                "no enforcement"
        true    | DEP_NAMES[3]   | NO_ENFORCEMENTS          | true                | 1         |
                "create enforce; no policy enforce"
    }

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("PZ")
    def "Verify attempted alerts on deployment updates: #desc"() {
        given:
        "Create deployment not violating 'Latest Tag' policy"
        assert orchestrator.createDeploymentNoWait(DEPLOYMENTS.get(DEP_NAMES[4]))

        when:
        "Set 'Latest Tag' policy enforcement to #policyEnforcements"
        Services.updatePolicyEnforcement(LATEST_TAG_POLICY_NAME, policyEnforcements, true)
        def policy = Services.getPolicyByName(LATEST_TAG_POLICY_NAME)
        assert policy && policy.getName() == LATEST_TAG_POLICY_NAME
        assert policy.enforcementActionsList == policyEnforcements

        and:
        "Set admission controller settings to enforce on updates to #enforce"
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(false)
                .setEnforceOnUpdates(enforce)
                .setTimeoutSeconds(3)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Sleep to allow settings update to propagate
        sleep(5000)

        and:
        "Trigger update deployment with latest tag"
        def cloned = DEPLOYMENTS.get(DEP_NAMES[4]).clone()
        cloned.setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest")
        def updated = orchestrator.updateDeploymentNoWait(cloned)

        then:
        "Verify deployment update"
        assert updated == updateShouldSucceed

        and:
        "Verify alerts"
        List<ListAlert> listAlerts = []
        withRetry(3, 3) {
            listAlerts = Services.getViolationsWithTimeout(DEP_NAMES[4], LATEST_TAG_POLICY_NAME, 60)
            // Expected number of alerts for deployment relies on the order of data inputs.
            assert listAlerts && listAlerts.size() == numAlerts
            assert listAlerts.get(0).getPolicy().getName() == LATEST_TAG_POLICY_NAME

            // Alerts are sorted in descending order of their violation time, therefore, the following check is
            // applied to most recent violations.
            if (updateShouldSucceed) {
                assert listAlerts.get(0).getState() == ViolationState.ACTIVE
            } else {
                assert listAlerts.get(0).getState() == ViolationState.ATTEMPTED
                // Verify admission controller enforcement action is applied.
                assert listAlerts.get(0).getEnforcementAction() == EnforcementAction.FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT
            }
        }

        // Verify that the alerts are not merged.
        assert AlertService.getViolation(listAlerts.get(0).getId()).getViolationsList().size() == 1

        where:
        "Data inputs are: "

        enforce | policyEnforcements       | updateShouldSucceed | numAlerts |
                desc
        true    | DEPLOY_TIME_ENFORCEMENTS | false                | 1         |
                "update enforce; policy enforce"
        // Attempted deploy-time alerts are not merged, hence, 2 attempted alerts expected.
        true    | DEPLOY_TIME_ENFORCEMENTS | false                | 2         |
                "update enforce; policy enforce; 2nd attempt"
        // 1 active and 2 attempted alerts are expected.
        false   | DEPLOY_TIME_ENFORCEMENTS | true                 | 3         |
                "no update enforce; policy enforce"
    }

    @Unroll
    @Tag("BAT")
    @Tag("RUNTIME")
    @Tag("PZ")
    // K8s event detection is currently not supported on OpenShift.
    @IgnoreIf({ Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT })
    def "Verify attempted alerts on kubernetes events: #desc"() {
        given:
        "Admission Controller exec/pf is enabled"
        assert ClusterService.getCluster().getAdmissionControllerEvents()

        and:
        "Create deployment"
        def dep = DEPLOYMENTS.get(DEP_NAMES[5])
        orchestrator.createDeployment(dep)
        assert Services.waitForDeployment(dep)

        when:
        "Set 'Exec into Pod' policy enforcement to #policyEnforcements"
        Services.updatePolicyEnforcement(KUBECTL_EXEC_POLICY_NAME, policyEnforcements, true)
        def policy = Services.getPolicyByName(KUBECTL_EXEC_POLICY_NAME)
        assert policy && policy.getName() == KUBECTL_EXEC_POLICY_NAME
        assert policy.enforcementActionsList == policyEnforcements
        // Sleep to allow settings update to propagate
        sleep(5000)

        and:
        "Exec into pod"
        def execed = orchestrator.execInContainer(dep, "ls -l")

        then:
        "Verify enforcement on exec"
        assert execed == execShouldSucceed

        and:
        "Verify alerts"
        List<ListAlert> listAlerts = []
        withRetry(3, 3) {
            listAlerts = Services.getViolationsWithTimeout(dep.name, KUBECTL_EXEC_POLICY_NAME, 60)
            assert listAlerts && listAlerts.size() == numAlerts
            assert listAlerts.get(0).getPolicy().getName() == KUBECTL_EXEC_POLICY_NAME

            // Alerts are sorted in descending order of their violation time, therefore, the following check is
            // applied to most recent violations.
            if (!execShouldSucceed) {
                assert listAlerts.get(0).getState() == ViolationState.ATTEMPTED
                // Verify admission controller enforcement action is applied.
                assert listAlerts.get(0).getEnforcementAction() == EnforcementAction.FAIL_KUBE_REQUEST_ENFORCEMENT
            }
        }

        // Verify that the alerts are not merged.
        assert AlertService.getViolation(listAlerts.get(0).getId()).getViolationsList().size() == numViolations

        where:
        "Data inputs are: "

        policyEnforcements    | execShouldSucceed | numAlerts | numViolations | desc
        RUN_TIME_ENFORCEMENTS | false             | 1         | 1             | "enforce"
        RUN_TIME_ENFORCEMENTS | false             | 1         | 2             | "enforce; 2nd attempt"
        NO_ENFORCEMENTS       | true              | 2         | 1             | "no enforcement"
        NO_ENFORCEMENTS       | true              | 2         | 2             | "no enforcement; 2nd attempt"
        RUN_TIME_ENFORCEMENTS | false             | 2         | 3             | "enforce; 3rd attempt"
        RUN_TIME_ENFORCEMENTS | false             | 2         | 4             | "enforce; 4th enforce"
    }
}
