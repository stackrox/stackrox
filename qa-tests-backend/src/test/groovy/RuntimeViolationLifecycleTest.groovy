import static Services.getPolicies
import static Services.getViolationsByDeploymentID
import static Services.roxDetectedDeployment
import static Services.updatePolicy
import static Services.updatePolicyToExclusionDeployment

import java.util.stream.Collectors

import io.stackrox.proto.storage.AlertOuterClass.ViolationState
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ProcessIndicatorOuterClass.ProcessIndicator

import objects.Deployment
import services.AlertService
import util.Timer

import spock.lang.Tag

class RuntimeViolationLifecycleTest extends BaseSpecification  {
    static final private String APTGETPOLICY = "Ubuntu Package Manager Execution"

    static final private String DEPLOYMENTNAME = "runtimeviolationlifecycle"
    static final private Deployment DEPLOYMENT = new Deployment()
        .setName(DEPLOYMENTNAME)
        .setImage ("quay.io/rhacs-eng/qa-multi-arch:nginx-" +
                "204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad")
        .addLabel ("app", DEPLOYMENTNAME)
        .setCommand(["sh" , "-c" , "apt-get -y update || true && sleep 600"])

    def checkPolicyExists(String policyName) {
        assert getPolicies().stream()
            .filter { f -> f.getName() == policyName }
            .collect(Collectors.toList()).size() == 1
    }

    def deleteAndWaitForSR(Deployment deployment) {
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)

        // Wait until the deployment disappears from StackRox.
        Timer t = new Timer(60, 1)
        boolean disappearedFromStackRox = false
        while (t.IsValid()) {
            if (!roxDetectedDeployment(deployment.getDeploymentUid(), deployment.getName())) {
                disappearedFromStackRox = true
                break
            }
        }
        return disappearedFromStackRox
    }

    def assertAlertExistsForDeploymentUidAndGetViolations(String policyName, String deploymentUid) {
        checkPolicyExists(APTGETPOLICY)
        def violations = Services.getViolationsByDeploymentID(deploymentUid, policyName, false, 66)
        assert !violations?.empty

        for (def violation : violations) {
            assert violation.getDeployment().getId() == deploymentUid
            assert violation.getLifecycleStage() == PolicyOuterClass.LifecycleStage.RUNTIME
            def alert = AlertService.getViolation(violation.getId())
            assert alert.getState() == ViolationState.ACTIVE
        }
        return violations
    }

/*
    TODO(ROX-3101)
    @Tag("BAT")
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
        assert violation.getLifecycleStage() == PolicyOuterClass.LifecycleStage.RUNTIME

        when:
        "Fetch the alert corresponding to the original apt-get violation"
        def alert = AlertService.getViolation(violation.getId())

        then:
        "Ensure the alert is active"
        assert alert.getState() == ViolationState.ACTIVE

        when:
        "Resolve the alert, get it again"
        AlertService.resolveAlert(alert.getId())
        sleep(1000)
        def resolvedAlert = AlertService.getViolation(alert.getId())

        then:
        "Ensure the alert is now resolved"
        assert resolvedAlert.getState() == ViolationState.RESOLVED

        cleanup:
        if (deploymentCreated) {
            orchestrator.deleteDeployment(DEPLOYMENT)
        }
    }
*/

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    
    def "Verify runtime excluded scope lifecycle"() {
        setup:
        "Create the deployment, verify that policy exists"

        orchestrator.createDeployment(DEPLOYMENT)
        boolean deploymentCreated = Services.waitForDeployment(DEPLOYMENT)

        assert deploymentCreated
        checkPolicyExists(APTGETPOLICY)

        // We update the apt-get policy in this test, and keep the original here so we can restore it.
        PolicyOuterClass.Policy originalAptGetPolicy = null

        when:
        "Get initial violations"
        def aptGetViolations = getViolationsByDeploymentID(DEPLOYMENT.getDeploymentUid(), APTGETPOLICY, false, 60)

        then:
        "Verify initial violation is triggered and has the properties we expect"
        // TODO(ROX-3577): Check that there is exactly one matching violation.
        assert !aptGetViolations?.empty
        def originalAptGetViolation = aptGetViolations.find {
            it.deployment.id == DEPLOYMENT.deploymentUid && it.lifecycleStage == PolicyOuterClass.LifecycleStage.RUNTIME
        }
        assert originalAptGetViolation : "Matching violation not found among ${aptGetViolations}"

        when:
        "Fetch the alert corresponding to the original apt-get violation"
        def originalAptGetAlert = AlertService.getViolation(originalAptGetViolation.getId())

        then:
        "Assert that the alert has the fields we expect"
        assert originalAptGetAlert != null
        assert originalAptGetAlert.getState() == ViolationState.ACTIVE
        assert originalAptGetAlert.getDeployment().getId() == DEPLOYMENT.getDeploymentUid()
        assert originalAptGetAlert.getLifecycleStage() == PolicyOuterClass.LifecycleStage.RUNTIME
        assert originalAptGetAlert.getProcessViolation() != null
        def processViolation = originalAptGetAlert.getProcessViolation()
        assert processViolation != null
        assert processViolation.getProcessesCount() > 0
        for (ProcessIndicator process : processViolation.getProcessesList()) {
            assert process.getSignal().getName() in ["apt-get", "dpkg", "apt"]
            if (process.getSignal().getName() == "apt-get") {
                assert process.getSignal().getArgs() == "-y update"
            }
        }

        when:
        "Exclude the deployment, get the alert again"
        originalAptGetPolicy = updatePolicyToExclusionDeployment(APTGETPOLICY, DEPLOYMENT)
        sleep(1000)
        def updatedAptGetAlert = AlertService.getViolation(originalAptGetViolation.getId())

        then:
        "Verify the alert is now resolved"
        assert updatedAptGetAlert != null
        assert updatedAptGetAlert.getState() == ViolationState.RESOLVED

        cleanup:
        if (deploymentCreated) {
            orchestrator.deleteAndWaitForDeploymentDeletion(DEPLOYMENT)
        }

        // Restore the original policy.
        if (originalAptGetPolicy != null) {
            updatePolicy(originalAptGetPolicy)
        }
    }

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    
    def "Verify runtime alert remains after deletion"() {
        setup:
        "Create the deployment, verify that policy exists"

        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT)

        def violations = assertAlertExistsForDeploymentUidAndGetViolations(APTGETPOLICY, DEPLOYMENT.getDeploymentUid())
        for (def violation: violations) {
            def alert = AlertService.getViolation(violation.getId())
            assert alert.getDeployment() != null && !alert.getDeployment().getInactive()
        }

        //// We delete the deployment in the middle of this test, but we keep this flag so that we know to clean up
        //// in case the test didn't make it that far.
        boolean deploymentDeleted

        when:
        "Delete the deployment, wait for it to disappear from StackRox, and fetch the new runtime alert."
        // Make sure the deployment initially exists, so that we know it's really gone when we check below.
        assert roxDetectedDeployment(DEPLOYMENT.getDeploymentUid(), DEPLOYMENT.getName())
        deploymentDeleted = deleteAndWaitForSR(DEPLOYMENT)

        then:
        assert deploymentDeleted
        def newViolations =
                assertAlertExistsForDeploymentUidAndGetViolations(APTGETPOLICY, DEPLOYMENT.getDeploymentUid())
        assert (newViolations*.id).toSet().containsAll(violations*.id)
        for (def violation: newViolations) {
            def alert = AlertService.getViolation(violation.getId())
            assert alert.getDeployment() != null && alert.getDeployment().getInactive()
        }

        cleanup:
        if (!deploymentDeleted) {
            orchestrator.deleteAndWaitForDeploymentDeletion(DEPLOYMENT)
        }
    }
}
