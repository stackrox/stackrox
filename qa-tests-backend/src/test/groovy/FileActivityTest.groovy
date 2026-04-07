import static util.Helpers.waitForTrue

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

    static final private String TEST_IMAGE =
            "quay.io/rhacs-eng/qa-multi-arch:nginx-" +
            "204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad"

    @Shared
    private final Deployment testDeployment = new Deployment()
            .setName("fa-test-${RUN_ID}")
            .setImage(TEST_IMAGE)
            .setCommand(["sh", "-c", "sleep 3600"])
            .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)

    def setupSpec() {
        Assume.assumeTrue(
                "FACT container not found in collector DaemonSet",
                orchestrator.containsDaemonSetContainer(
                        Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER))

        setFactEnv("/tmp/**/*", true)

        orchestrator.createDeployment(testDeployment)
        assert Services.waitForDeployment(testDeployment)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(testDeployment)
        setFactEnv("", false)
    }

    @Tag("BAT")
    def "Verify deployment-level file activity alert is triggered"() {
        given:
        "a file activity policy for a unique path"
        def path = "/tmp/fa-deploy-${RUN_ID}"
        def policyName = "FA-E2E-deploy-${RUN_ID}"
        def policy = createFileActivityPolicy(
                policyName, path,
                PolicyOuterClass.EventSource.DEPLOYMENT_EVENT, "CREATE")
        def policyID = PolicyService.createNewPolicy(policy)
        assert policyID

        when:
        "a file is created at that path inside the deployment"
        assert orchestrator.execInContainer(testDeployment, "touch ${path}")

        then:
        "an alert is triggered"
        assert Services.waitForViolation(testDeployment.name, policyName, 90)

        and:
        "the alert contains file access violation details"
        def violations = AlertService.getViolations(
                ListAlertsRequest.newBuilder()
                        .setQuery("Deployment:${testDeployment.name}+Policy:${policyName}")
                        .build())
        assert violations.size() >= 1

        def alert = AlertService.getViolation(violations[0].id)
        assert alert.violationsList.size() > 0
        assert alert.violationsList[0].type == AlertOuterClass.Alert.Violation.Type.FILE_ACCESS
        assert alert.violationsList[0].message.contains(path)

        cleanup:
        if (policyID) {
            PolicyService.deletePolicy(policyID)
        }
    }

    @Tag("BAT")
    def "Verify node-level file activity alert is triggered"() {
        given:
        "a file activity policy for a unique path with node event source"
        def path = "/tmp/fa-node-${RUN_ID}"
        def policyName = "FA-E2E-node-${RUN_ID}"
        def policy = createFileActivityPolicy(
                policyName, path,
                PolicyOuterClass.EventSource.NODE_EVENT, "CREATE")
        def policyID = PolicyService.createNewPolicy(policy)
        assert policyID

        and:
        "a privileged deployment that can exec on the host"
        def hostDeployment = new Deployment()
                .setName("fa-host-${RUN_ID}")
                .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-33-1")
                .setCommand(["sh", "-c", "sleep 3600"])
                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                .setPrivilegedFlag(true)
                .addHostMount("host-root", "/host")
        orchestrator.createDeployment(hostDeployment)
        assert Services.waitForDeployment(hostDeployment)

        when:
        "a file is created on the host via chroot"
        // sudo is needed to trigger a node-level file event; without it the
        // access is attributed to the container and no node alert fires.
        assert orchestrator.execInContainer(hostDeployment, "chroot /host sudo touch ${path}")

        then:
        "a node-level alert is triggered"
        assert Services.waitForNodeViolation(policyName, 90)

        cleanup:
        if (hostDeployment) {
            orchestrator.execInContainer(hostDeployment, "chroot /host sudo rm -f ${path}")
            orchestrator.deleteDeployment(hostDeployment)
        }
        if (policyID) {
            PolicyService.deletePolicy(policyID)
        }
    }

    @CompileStatic
    private void setFactEnv(String paths, boolean json) {
        String jsonStr = Boolean.toString(json)
        log.info "Setting FACT env on collector DaemonSet: FACT_PATHS=${paths}, FACT_JSON=${jsonStr}"

        orchestrator.updateDaemonSetEnv(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                "FACT_PATHS", paths)
        orchestrator.updateDaemonSetEnv(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                "FACT_JSON", jsonStr)

        log.info "Waiting for collector DS to pick up FACT env vars and be ready"
        waitForTrue(20, 10) {
            orchestrator.daemonSetEnvVarUpdated(
                    Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                    "FACT_PATHS", paths) &&
            orchestrator.daemonSetEnvVarUpdated(
                    Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                    "FACT_JSON", jsonStr) &&
            orchestrator.daemonSetReady(Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS)
        }
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
}
