import static Services.getPolicies
import static Services.getViolations

import com.google.gson.GsonBuilder
import org.junit.Test
import objects.PolicyResults
import objects.AlertsByPolicy
import com.google.gson.Gson

class SystemPoliciesTest extends  BaseSpecification  {

    @Test
    def "Verify custom policy Latest tag can be added and Violation is triggered: C811"() {
        String indexIP = System.getenv("index")
        List<String>violations = new ArrayList<>()
        String deployName = "qaport22"

        when:
        "Pull image Port 22"
        orchestrator.setDeploymentName(deployName)
        orchestrator.addContainerPort(22)
        assert orchestrator.createDeployment()

        and:
        "Policy is available to trigger the violations"
        Gson gson = new GsonBuilder().create()
        def res = getPolicies(indexIP)
        PolicyResults jsonResultRule = gson.fromJson(res, PolicyResults)
        assert jsonResultRule.policies.name.contains("Container Port 22")

        then:
        "Verify Violations have been triggered"
        def result = getViolations(indexIP)
        AlertsByPolicy jsonResultViolations = gson.fromJson(result, AlertsByPolicy)
        for (int i = 0; i < jsonResultViolations.alertsByPolicies.size(); i++) {
            violations.add(jsonResultViolations.alertsByPolicies[i].policy.name)
        }
        assert violations.contains("Container Port 22")

        cleanup:
        "Remove Deployment"
        def teststatus = tc.verifyAndAdd(violations, "Container Port 22", 811)
        resultMap.put(811, teststatus)
        orchestrator.deleteDeployment(deployName)
    }

}
