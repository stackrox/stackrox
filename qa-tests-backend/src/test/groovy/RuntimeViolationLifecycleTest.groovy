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
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.PolicyServiceOuterClass

class RuntimeViolationLifecycleTest extends BaseSpecification  {
    static final private String APTGETPOLICY = "Ubuntu Package Manager Execution"

    static final private String DEPLOYMENTNAME = "runtimeviolationlifecycle"
    static final private Deployment DEPLOYMENT = new Deployment()
        .setName(DEPLOYMENTNAME)
        .setImage ("nginx@sha256:9ad0746d8f2ea6df3a17ba89eca40b48c47066dfab55a75e08e2b70fc80d929e")
        .addLabel ("app", DEPLOYMENTNAME)
        .setCommand(["sh" , "-c" , "apt-get -y update && sleep 600"])

    def checkPolicyExists(String policyName) {
        assert getPolicies().stream()
            .filter { f -> f.getName() == policyName }
            .collect(Collectors.toList()).size() == 1
    }

    def deleteAndWaitForSR(Deployment deployment) {
        orchestrator.deleteDeployment(deployment)

        // Wait until the deployment disappears from StackRox.
        long sleepTime = 0
        long sleepInterval = 1000
        boolean disappearedFromStackRox = false
        while (sleepTime < 60000) {
            if (!roxDetectedDeployment(deployment.getDeploymentUid())) {
                disappearedFromStackRox = true
                break
            }
            sleep(sleepInterval)
            sleepTime += sleepInterval
        }
        return disappearedFromStackRox
    }

    def assertAlertExistsForDeploymentUid(String policyName, String deploymentUid) {
        checkPolicyExists(APTGETPOLICY)
        def violations = getViolationsByDeploymentID(deploymentUid, policyName, 60)
        assert violations?.size() == 1
        def violation = violations[0]
        assert violation.getDeployment().getId() == deploymentUid
        assert violation.getLifecycleStage() == PolicyServiceOuterClass.LifecycleStage.RUNTIME
        def alert = getViolation(violation.getId())
        assert alert.getState() == AlertServiceOuterClass.ViolationState.ACTIVE
        return true
    }

    @Category(BAT)
    def "Verify runtime resolution lifecycle"() {
        setup:
        "Create the deployment, verify that policy exists"

        orchestrator.createDeployment(DEPLOYMENT)
        boolean deploymentCreated = Services.waitForDeployment(DEPLOYMENT)

        assert deploymentCreated
        checkPolicyExists(APTGETPOLICY)

        when:
        "Get initial violations"
        def violations = getViolationsByDeploymentID(DEPLOYMENT.getDeploymentUid(), APTGETPOLICY, 60)

        then:
        "Verify initial violation is triggered and has the properties we expect"
        assert violations?.size() == 1
        def violation = violations[0]
        assert violation.getDeployment().getId() == DEPLOYMENT.getDeploymentUid()
        assert violation.getLifecycleStage() == PolicyServiceOuterClass.LifecycleStage.RUNTIME

        when:
        "Fetch the alert corresponding to the original apt-get violation"
        def alert = getViolation(violation.getId())

        then:
        "Ensure the alert is active"
        assert alert.getState() == AlertServiceOuterClass.ViolationState.ACTIVE

        when:
        "Resolve the alert, get it again"
        resolveAlert(alert.getId())
        sleep(1000)
        def resolvedAlert = getViolation(alert.getId())

        then:
        "Ensure the alert is now resolved"
        assert resolvedAlert.getState() == AlertServiceOuterClass.ViolationState.RESOLVED

        cleanup:
        if (deploymentCreated) {
            orchestrator.deleteDeployment(DEPLOYMENT)
        }
    }

    @Category(BAT)
    def "Verify runtime whitelist lifecycle"() {
        setup:
        "Create the deployment, verify that policy exists"

        orchestrator.createDeployment(DEPLOYMENT)
        boolean deploymentCreated = Services.waitForDeployment(DEPLOYMENT)

        assert deploymentCreated
        checkPolicyExists(APTGETPOLICY)

        // We update the apt-get policy in this test, and keep the original here so we can restore it.
        PolicyServiceOuterClass.Policy originalAptGetPolicy = null

        when:
        "Get initial violations"
        def aptGetViolations = getViolationsByDeploymentID(DEPLOYMENT.getDeploymentUid(), APTGETPOLICY, 60)

        then:
        "Verify initial violation is triggered and has the properties we expect"
        assert aptGetViolations?.size() == 1
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
        def violatingProcess = subViolation.getProcessesList().
            find { p -> p.getSignal().getName() == "apt-get" }
        assert violatingProcess != null
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

        cleanup:
        if (deploymentCreated) {
            orchestrator.deleteDeployment(DEPLOYMENT)
        }

        // Restore the original policy.
        if (originalAptGetPolicy != null) {
            updatePolicy(originalAptGetPolicy)
        }
    }

    @Category(BAT)
    def "Verify runtime alert remains after deletion"() {
        setup:
        "Create the deployment, verify that policy exists"

        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT)

        assertAlertExistsForDeploymentUid(APTGETPOLICY, DEPLOYMENT.getDeploymentUid())

        //// We delete the deployment in the middle of this test, but we keep this flag so that we know to clean up
        //// in case the test didn't make it that far.
        boolean deploymentDeleted = false

        when:
        "Delete the deployment, wait for it to disappear from StackRox, and fetch the new runtime alert."
        // Make sure the deployment initially exists, so that we know it's really gone when we check below.
        assert roxDetectedDeployment(DEPLOYMENT.getDeploymentUid())
        deploymentDeleted = deleteAndWaitForSR(DEPLOYMENT)

        then:
        assert deploymentDeleted
        assert assertAlertExistsForDeploymentUid(APTGETPOLICY, DEPLOYMENT.getDeploymentUid())

        cleanup:
        if (!deploymentDeleted) {
            orchestrator.deleteDeployment(DEPLOYMENT)
        }
    }
}
