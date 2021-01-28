import groups.BAT
import groups.K8sEvents
import groups.RUNTIME
import io.stackrox.proto.storage.AlertOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import objects.Deployment
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.AlertService
import services.FeatureFlagService
import spock.lang.Retry
import spock.lang.Unroll
import util.Env

class K8sEventDetectionTest extends BaseSpecification {
    static final private List<Deployment> DEPLOYMENTS = []

    static private registerDeployment(String name, boolean privileged) {
        DEPLOYMENTS.add(
            new Deployment().setName(name).setImage("nginx:1.14-alpine").addLabel("app", name).
                setPrivilegedFlag(privileged)
        )
        return name
    }

    // Deployment names
    static final private String NGINX_1_DEP_NAME = registerDeployment("k8seventnginx1", false)
    static final private String NGINX_2_DEP_NAME = registerDeployment("k8seventnginx2", false)
    static final private String PRIV_NGINX_1_DEPNAME = registerDeployment("k8seventprivnginx1", true)
    static final private String PRIV_NGINX_2_DEPNAME = registerDeployment("k8seventprivnginx2", true)

    static final private String KUBECTL_EXEC_POLICY_NAME = "Kubernetes Actions: Exec into Pod"

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanupSpec() {
        for (def deployment: DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    def runExec(List<Deployment> deployments) {
        for (def deployment: deployments) {
            assert orchestrator.execInContainer(deployment, "ls -l")
        }
        return true
    }

    def checkViolationsAreAsExpected(List<String> execedIntoDeploymentNames, List<String> violatingDeploymentNames,
                       Map<String, String> podNames, int expectedK8sViolationsCount) {
        for (def violatingDeploymentName: violatingDeploymentNames) {
            def violatingDeployment = DEPLOYMENTS.find { it.name == violatingDeploymentName }
            assert violatingDeployment
            def violations = Services.getViolationsByDeploymentID(
                violatingDeployment.deploymentUid, KUBECTL_EXEC_POLICY_NAME, 60)
            assert violations != null && violations.size() == 1
            def fullViolation = AlertService.getViolation(violations.get(0).getId())
            assert fullViolation
            print "Violation for ${violatingDeploymentName} while checking for" +
                "${expectedK8sViolationsCount} violations: ${fullViolation}"
            def k8sSubViolations = fullViolation.getViolationsList().findAll {
                it.getType() == AlertOuterClass.Alert.Violation.Type.K8S_EVENT
            }
            def podName = podNames.get(violatingDeploymentName)
            assert k8sSubViolations.size() == expectedK8sViolationsCount
            for (def subViolation: k8sSubViolations) {
                assert subViolation.message == "Kubernetes API received exec 'ls -l' request into pod '${podName}'"
                def kvAttrs = subViolation.getKeyValueAttrs().getAttrsList()
                def podAttr = kvAttrs.find { it.key == "pod" }
                assert podAttr != null && podAttr.value == podName
                def commandsAttr = kvAttrs.find { it.key == "commands" }
                assert commandsAttr != null && commandsAttr.value == "ls -l"
            }

            // Ensure the deployment enrichment works.
            def deploymentFromViolation = fullViolation.getDeployment()
            assert deploymentFromViolation != null && deploymentFromViolation.getId() ==
                violatingDeployment.deploymentUid
        }

        for (def deploymentName: execedIntoDeploymentNames) {
            if (violatingDeploymentNames.any { it == deploymentName }) {
                continue
            }
            println "Checking that deployment ${deploymentName} does NOT have a violation"
            def deployment = DEPLOYMENTS.find { it.name == deploymentName }
            assert deployment
            assert Services.checkForNoViolationsByDeploymentID(deployment.deploymentUid, KUBECTL_EXEC_POLICY_NAME)
        }
        return true
    }

    @Retry(count = 0)
    @Unroll
    @Category([BAT, RUNTIME, K8sEvents])
    def "Verify k8s exec detection into #execIntoDeploymentNames with addl groups #additionalPolicyGroups"() {
        when:
        "Create the deployments, modify the policy, exec into them"
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_K8S_EVENTS_DETECTION"))
        // K8s event detection is currently not supported on OpenShift.
        Assume.assumeTrue(Env.mustGetOrchestratorType() != OrchestratorTypes.OPENSHIFT)

        def originalPolicy = Services.getPolicyByName(KUBECTL_EXEC_POLICY_NAME)
        assert originalPolicy != null && originalPolicy.getName() == KUBECTL_EXEC_POLICY_NAME

        def currentPolicy = originalPolicy
        if (additionalPolicyGroups != null && additionalPolicyGroups.size() > 0) {
            assert originalPolicy.getPolicySectionsCount() == 1
            def policySection = originalPolicy.getPolicySections(0)
            def newPolicySection = PolicyOuterClass.PolicySection.newBuilder(policySection).
                addAllPolicyGroups(additionalPolicyGroups).
                build()
            currentPolicy = PolicyOuterClass.Policy.newBuilder(originalPolicy).
                clearPolicySections().
                addPolicySections(newPolicySection).
                build()
            Services.updatePolicy(currentPolicy)
            // Sleep to allow policy update to propagate
            sleep(3000)
        }

        def podNames = new HashMap<String, String>()
        def execIntoDeployments = []
        for (def deploymentName: execIntoDeploymentNames) {
            def deployment = DEPLOYMENTS.find { it.name == deploymentName }
            assert deployment
            execIntoDeployments.add(deployment)

            def podsForDeployment = orchestrator.getPods(deployment.namespace, deployment.getLabels()["app"])
            assert podsForDeployment != null && podsForDeployment.size() == 1
            podNames.put(deployment.name, podsForDeployment.get(0).metadata.name)
        }

        assert runExec(execIntoDeployments)

        then:
        "Fetch violations and assert on properties"
        assert checkViolationsAreAsExpected(execIntoDeploymentNames, violatingDeploymentNames, podNames, 1)

        when:
        "Run another exec"
        assert runExec(execIntoDeployments)

        then:
        "Violations should have the new exec appended to them"
        withRetry(2, 3) {
            assert checkViolationsAreAsExpected(execIntoDeploymentNames, violatingDeploymentNames, podNames, 2)
        }

        when:
        "Update the policy to have enforcement"
        currentPolicy = PolicyOuterClass.Policy.newBuilder(currentPolicy)
            .clearEnforcementActions()
            .addEnforcementActions(PolicyOuterClass.EnforcementAction.FAIL_KUBE_REQUEST_ENFORCEMENT)
            .build()
        Services.updatePolicy(currentPolicy)
        // Allow to propagate
        sleep(3000)

        then:
        "Exec should fail for all violating deployments, but not for the others, and violations should not be updated"
        for (def deploymentName: execIntoDeploymentNames) {
            def execShouldSucceed = (violatingDeploymentNames.find { it == deploymentName } == null)
            def deployment = DEPLOYMENTS.find { it.name == deploymentName }
            assert deployment
            assert orchestrator.execInContainer(deployment, "ls -l") == execShouldSucceed
        }

        // Still only 2 k8s violations since the updates were blocked
        assert checkViolationsAreAsExpected(execIntoDeploymentNames, violatingDeploymentNames, podNames, 2)

        cleanup:
        Services.updatePolicy(originalPolicy)

        where:
        "Data inputs are"
        additionalPolicyGroups | execIntoDeploymentNames | violatingDeploymentNames

        [] | [NGINX_1_DEP_NAME, PRIV_NGINX_1_DEPNAME] | [NGINX_1_DEP_NAME, PRIV_NGINX_1_DEPNAME]
        [PolicyOuterClass.PolicyGroup.newBuilder().
            setFieldName("Privileged Container").
            addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("true").build()).
            build(),] | [NGINX_2_DEP_NAME, PRIV_NGINX_2_DEPNAME] | [PRIV_NGINX_2_DEPNAME]
    }
}
