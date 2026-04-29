import static util.FileActivityUtil.createFileActivityPolicy
import static util.FileActivityUtil.isFactAvailable
import static util.FileActivityUtil.removeFactEnv
import static util.FileActivityUtil.resolveAlertsByPolicy
import static util.FileActivityUtil.setFactEnv

import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.storage.AlertOuterClass
import io.stackrox.proto.storage.PolicyOuterClass

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
                isFactAvailable(orchestrator))

        setFactEnv(orchestrator, "/tmp/**/*", true)

        orchestrator.createDeployment(testDeployment)
        assert Services.waitForDeployment(testDeployment)
    }

    def cleanupSpec() {
        if (isFactAvailable(orchestrator)) {
            orchestrator.deleteDeployment(testDeployment)
            removeFactEnv(orchestrator)
        }
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
        resolveAlertsByPolicy(policyName)
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
        resolveAlertsByPolicy(policyName)
        if (policyID) {
            PolicyService.deletePolicy(policyID)
        }
    }

}
