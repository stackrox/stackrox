import groups.BAT
import groups.K8sEvents
import groups.RUNTIME
import objects.Deployment
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.AlertService
import services.FeatureFlagService
import util.Env

class K8sEventDetectionTest extends BaseSpecification {
    // Deployment names
    static final private String NGINXDEPLOYMENT = "qanginx"

    static final private String KUBECTL_EXEC_POLICY_NAME = "Kubectl Exec into Pod"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                .setName (NGINXDEPLOYMENT)
                .setImage ("nginx:1.14-alpine")
                .addLabel("app", NGINXDEPLOYMENT),
     ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Category([BAT, RUNTIME, K8sEvents])
    def "Verify k8s exec detection"()  {
        when:
        "Get nginx deployment and pod, get kubectl exec policy, exec into pod"
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_K8S_EVENTS_DETECTION"))

        // K8s event detection is currently not supported on OpenShift.
        Assume.assumeTrue(Env.mustGetOrchestratorType() != OrchestratorTypes.OPENSHIFT)

        def nginxDeployment = DEPLOYMENTS.find { it.name == NGINXDEPLOYMENT }
        assert nginxDeployment != null

        def policy = Services.getPolicyByName(KUBECTL_EXEC_POLICY_NAME)
        assert policy != null && policy.getName() == KUBECTL_EXEC_POLICY_NAME

        def pods = orchestrator.getPods(nginxDeployment.namespace, NGINXDEPLOYMENT)
        assert pods != null && pods.size() == 1
        def pod = pods.get(0)

        assert orchestrator.execInContainer(nginxDeployment, "ls -l")

        then:
        "Fetch violation, and assert on its properties"
        def violations = Services.getViolationsByDeploymentID(
            nginxDeployment.deploymentUid, KUBECTL_EXEC_POLICY_NAME, 60)
        assert violations != null && violations.size() == 1
        def fullViolation = AlertService.getViolation(violations.get(0).getId())
        print "Violation: ${fullViolation}"
        assert fullViolation.getViolationsCount() == 1
        def subViolation = fullViolation.getViolations(0)
        // TODO(Mandar): Update these messages when we remove the comma separation
        assert subViolation.message == "Kubernetes API received exec 'ls, -l' request into pod '${pod.metadata.name}'"
        def kvAttrs = subViolation.getKeyValueAttrs().getAttrsList()
        def podAttr = kvAttrs.find { it.key == "pod" }
        assert podAttr != null && podAttr.value == pod.metadata.name
        def commandsAttr = kvAttrs.find { it.key == "commands" }
        assert commandsAttr != null && commandsAttr.value == "ls, -l"

        // Ensure the deployment enrichment works.
        def deploymentFromViolation = fullViolation.getDeployment()
        assert deploymentFromViolation != null && deploymentFromViolation.getId() == nginxDeployment.deploymentUid
    }
}
