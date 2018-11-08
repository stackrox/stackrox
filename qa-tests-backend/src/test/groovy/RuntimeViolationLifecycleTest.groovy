import static Services.getPolicies
import static Services.getViolation
import static Services.getViolationsByDeploymentID
import static Services.resolveAlert
import static Services.roxDetectedDeployment
import static Services.updatePolicy
import static Services.updatePolicyToWhitelistDeployment

import groups.BAT
import java.util.stream.Collectors
import objects.Deployment
import org.junit.experimental.categories.Category
import stackrox.generated.AlertServiceOuterClass
import stackrox.generated.PolicyServiceOuterClass

class RuntimeViolationLifecycleTest extends BaseSpecification  {
    static final private String APTPOLICY = "apt Execution"
    static final private String APTGETPOLICY = "apt-get Execution"
    static final private String DPKGPOLICY = "dpkg Execution"

    static final private String DEPLOYMENTNAME = "runtimeviolationlifecycle"
    static final private Deployment DEPLOYMENT = new Deployment()
        .setName(DEPLOYMENTNAME)
        .setImage ("nginx@sha256:9ad0746d8f2ea6df3a17ba89eca40b48c47066dfab55a75e08e2b70fc80d929e")
        .addLabel ("app", DEPLOYMENTNAME)
        .setCommand(["sh" , "-c" , "apt-get -y update && apt update && sleep 600"])

    @Category(BAT)
    def "Verify runtime alert lifecycle"() {
        setup:
        "Create the deployment, verify that policy exists"

        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT)
        for (String policyName: [APTPOLICY, APTGETPOLICY, DPKGPOLICY]) {
            assert getPolicies().stream()
                .filter { f -> f.getName() == policyName }
                .collect(Collectors.toList()).size() == 1
        }

        // We update the apt-get policy in this test, and keep the original here so we can restore it.
        PolicyServiceOuterClass.Policy originalAptGetPolicy = null

        // We delete the deployment in the middle of this test, but we keep this flag so that we know to clean up
        // in case the test didn't make it that far.
        boolean deploymentDeleted = false

        when:
        "Get initial violations"
        def aptGetViolations = getViolationsByDeploymentID(DEPLOYMENT.getDeploymentUid(), APTGETPOLICY, 60)
        def dpkgViolations = getViolationsByDeploymentID(DEPLOYMENT.getDeploymentUid(), DPKGPOLICY, 60)
        def aptViolations = getViolationsByDeploymentID(DEPLOYMENT.getDeploymentUid(), APTPOLICY, 60)

        then:
        "Verify initial violation is triggered and has the properties we expect"
        assert dpkgViolations?.size() == 1
        assert aptGetViolations?.size() == 1
        assert aptViolations?.size() == 1
        def originalAptGetViolation = aptGetViolations[0]
        assert originalAptGetViolation.getDeployment().getId() == DEPLOYMENT.getDeploymentUid()
        assert originalAptGetViolation.getLifecycleStage() == PolicyServiceOuterClass.LifecycleStage.RUNTIME

        when:
        "Fetch the alert corresponding to the original apt-get violation"
        def originalAptGetAlert = getViolation(originalAptGetViolation.getId())

        then:
        "Assert that the alert has the fields we expect"
        assert originalAptGetAlert != null
        assert originalAptGetAlert.getState() == AlertServiceOuterClass.ViolationState.ACTIVE
        assert originalAptGetAlert.getDeployment().getId() == DEPLOYMENT.getDeploymentUid()
        assert originalAptGetAlert.getLifecycleStage() == PolicyServiceOuterClass.LifecycleStage.RUNTIME
        assert originalAptGetAlert.getViolationsCount() == 1
        def subViolation = originalAptGetAlert.getViolations(0)
        assert subViolation.getProcessesCount() > 0
        def violatingProcess = subViolation.getProcesses(0)
        assert violatingProcess.getSignal().getName() == "apt-get"
        assert violatingProcess.getSignal().getArgs() == "-y update"

        when:
        "Whitelist the deployment, get the alert again"
        originalAptGetPolicy = updatePolicyToWhitelistDeployment(APTGETPOLICY, DEPLOYMENT)
        sleep(1000)
        def updatedAptGetAlert = getViolation(originalAptGetViolation.getId())

        then:
        "Verify the alert is now resolved"
        assert updatedAptGetAlert != null
        assert updatedAptGetAlert.getState() == AlertServiceOuterClass.ViolationState.RESOLVED

        when:
        "Fetch the alert corresponding to the dpkg violation"
        def originalDpkgAlert = getViolation(dpkgViolations.get(0).getId())

        then:
        "Ensure the alert is active"
        assert originalDpkgAlert.getState() == AlertServiceOuterClass.ViolationState.ACTIVE

        when:
        "Resolve the alert, get it again"
        resolveAlert(originalDpkgAlert.getId())
        sleep(1000)
        def updatedDpkgAlert = getViolation(originalDpkgAlert.getId())

        then:
        "Ensure the alert is now resolved"
        assert updatedDpkgAlert.getState() == AlertServiceOuterClass.ViolationState.RESOLVED

        when:
        "Fetch the alert corresponding to the apt violation"
        def originalAptAlert = getViolation(aptViolations.get(0).getId())

        then:
        "Ensure the alert is active"
        assert originalAptAlert.getState() == AlertServiceOuterClass.ViolationState.ACTIVE

        when:
        "Delete the deployment, wait for it to disappear from StackRox, and fetch the new runtime alert"
        // Make sure the deployment initially exists, so that we know it's really gone when we check below.
        assert roxDetectedDeployment(DEPLOYMENT.getDeploymentUid())

        orchestrator.deleteDeployment(DEPLOYMENT)

        // Wait until the deployment disappears from StackRox.
        long sleepTime = 0
        long sleepInterval = 1000
        boolean disappearedFromStackRox = false
        while (sleepTime < 60000) {
            if (!roxDetectedDeployment(DEPLOYMENT.getDeploymentUid())) {
                disappearedFromStackRox = true
                break
            }
            sleep(sleepInterval)
            sleepTime += sleepInterval
        }
        assert disappearedFromStackRox

        deploymentDeleted = true
        def updatedAptAlert = getViolation(aptViolations.get(0).getId())

        then:
        "the runtime alert is still active"
        assert updatedAptAlert.getState() == AlertServiceOuterClass.ViolationState.ACTIVE

        cleanup:
        if (!deploymentDeleted) {
            orchestrator.deleteDeployment(DEPLOYMENT)
        }

        // Restore the original policy.
        if (originalAptGetPolicy != null) {
            updatePolicy(originalAptGetPolicy)
        }
    }

}
