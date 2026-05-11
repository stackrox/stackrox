import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.storage.AlertOuterClass
import io.stackrox.proto.storage.PolicyOuterClass

import groovy.transform.CompileStatic
import objects.Deployment
import org.junit.Assume

import common.Constants
import services.AlertService
import services.PolicyService

import spock.lang.Shared
import spock.lang.Tag

@Tag("PZ")
class FileActivityTest extends BaseSpecification {

    static final private String DEPLOY_PATH = "/tmp/fa-deploy-${RUN_ID}"
    static final private String NODE_PATH = "/tmp/fa-node-${RUN_ID}"
    static final private String POLICY_WILDCARD = "/tmp/fa-*"
    static final private String DEPLOY_POLICY_NAME = "FA-E2E-deploy-${RUN_ID}"
    static final private String NODE_POLICY_NAME = "FA-E2E-node-${RUN_ID}"

    @Shared
    private String deployPolicyID

    @Shared
    private String nodePolicyID

    @Shared
    private final Deployment deployDeployment = new Deployment()
            .setName("fa-deploy-${RUN_ID}")
            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
            .setCommand(["/bin/sh", "-c",])
            .setArgs(["while sleep 1; do touch ${DEPLOY_PATH}; done" as String,])
            .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)

    @Shared
    private final Deployment nodeDeployment = new Deployment()
            .setName("fa-node-${RUN_ID}")
            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
            .setCommand(["/bin/sh", "-c",])
            .setArgs(["while sleep 1; do chroot /host sudo touch ${NODE_PATH}; done" as String,])
            .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
            .setPrivilegedFlag(true)
            .addHostMount("host-root", "/host")

    def setupSpec() {
        Assume.assumeTrue(
                "FACT container not found in collector DaemonSet",
                orchestrator.containsDaemonSetContainer(
                        Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER))

        deployPolicyID = PolicyService.createNewPolicy(createFileActivityPolicy(
                DEPLOY_POLICY_NAME, POLICY_WILDCARD,
                PolicyOuterClass.EventSource.DEPLOYMENT_EVENT, "OPEN"))
        assert deployPolicyID

        nodePolicyID = PolicyService.createNewPolicy(createFileActivityPolicy(
                NODE_POLICY_NAME, POLICY_WILDCARD,
                PolicyOuterClass.EventSource.NODE_EVENT, "OPEN"))
        assert nodePolicyID

        orchestrator.createDeployment(deployDeployment)
        orchestrator.createDeployment(nodeDeployment)
        assert Services.waitForDeployment(deployDeployment)
        assert Services.waitForDeployment(nodeDeployment)
    }

    def cleanupSpec() {
        if (orchestrator.containsDaemonSetContainer(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER)) {
            orchestrator.deleteDeployment(deployDeployment)
            orchestrator.deleteDeployment(nodeDeployment)
            resolveAlertsByPolicy(DEPLOY_POLICY_NAME)
            resolveAlertsByPolicy(NODE_POLICY_NAME)
            if (deployPolicyID) {
                PolicyService.deletePolicy(deployPolicyID)
            }
            if (nodePolicyID) {
                PolicyService.deletePolicy(nodePolicyID)
            }
        }
    }

    @Tag("BAT")
    def "Verify deployment-level file activity alert is triggered"() {
        expect:
        "an alert is triggered"
        assert Services.waitForViolation(deployDeployment.name, DEPLOY_POLICY_NAME, 90)

        and:
        "the alert contains file access violation details"
        def violations = AlertService.getViolations(
                ListAlertsRequest.newBuilder()
                        .setQuery("Policy:${DEPLOY_POLICY_NAME}")
                        .build())
        assert violations.size() >= 1

        def alert = AlertService.getViolation(violations[0].id)
        assert alert.violationsList.size() > 0
        assert alert.violationsList[0].type == AlertOuterClass.Alert.Violation.Type.FILE_ACCESS
        assert alert.violationsList[0].message.contains(DEPLOY_PATH)
    }

    @Tag("BAT")
    def "Verify node-level file activity alert is triggered"() {
        expect:
        "a node-level alert is triggered"
        assert Services.waitForNodeViolation(NODE_POLICY_NAME, 90)
    }

    @CompileStatic
    private static PolicyOuterClass.Policy createFileActivityPolicy(
            String name, String path, PolicyOuterClass.EventSource eventSource, String... operations) {
        def groups = [
                PolicyOuterClass.PolicyGroup.newBuilder()
                        .setFieldName("File Path")
                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(path))
                        .build(),
        ]

        if (operations.length > 0) {
            def opGroup = PolicyOuterClass.PolicyGroup.newBuilder()
                    .setFieldName("File Operation")
            operations.each { op ->
                opGroup.addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(op))
            }
            groups << opGroup.build()
        }

        return PolicyOuterClass.Policy.newBuilder()
                .setName(name)
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.RUNTIME)
                .setEventSource(eventSource)
                .setSeverityValue(2)
                .addCategories("File Activity Monitoring")
                .setDisabled(false)
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder()
                                .setSectionName("file-access")
                                .addAllPolicyGroups(groups)
                                .build()
                )
                .build()
    }

    private static void resolveAlertsByPolicy(String policyName) {
        def alerts = AlertService.getViolations(
                ListAlertsRequest.newBuilder()
                        .setQuery("Policy:${policyName}+Violation State:ACTIVE")
                        .build())
        for (alert in alerts) {
            AlertService.resolveAlert(alert.id)
        }
    }
}
