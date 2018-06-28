import static Services.getPolicies
import static Services.waitForViolation
import static Services.waitForDeployment

import objects.Deployment
import org.junit.Test
import java.util.stream.Collectors

class SystemPoliciesTest extends  BaseSpecification  {

    @Test
    def "Verify custom policy Port 22 can be added and Violation is triggered: C811"() {
        List<String>violations = new ArrayList<>()
        String deploymentName =  "qaport22"

        Deployment deployment = new Deployment()
                .setName(deploymentName)
                .setImage("nginx")
                .addPort(22)
                .addLabel("app", "test")

        when:
        "Create image with Port 22 exposed"
        orchestrator.createDeployment(deployment)

        and:
        "Policy is available to trigger the violations"
        assert getPolicies().stream()
                .filter { f -> f.getName() == "Container Port 22" }
                .collect(Collectors.toList()).size() == 1

        and:
        "Deployment has been registered"
        assert waitForDeployment(deploymentName)

        then:
        "Verify Violations have been triggered"
        assert waitForViolation(deploymentName, "Container Port 22", 20)

        cleanup:
        "Remove Deployment"
        def teststatus = tc.verifyAndAdd(violations, "Container Port 22", 811)
        resultMap.put(811, teststatus)
        orchestrator.deleteDeployment(deploymentName)
    }

}
