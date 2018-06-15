import OrchestratorManager.OrchestratorMain
import OrchestratorManager.OrchestratorType
import OrchestratorManager.OrchestratorTypes
import com.google.gson.GsonBuilder
import static Services.getPolicies
import static Services.getViolations
import Objects.PolicyResults
import Objects.AlertsByPolicy
import com.google.gson.Gson
import testrailIntegration.TestRail
import testrailIntegration.TestRailconfig
import junit.framework.TestFailure

class SystemPoliciesTest extends  BaseSpecification  {

    def "Verify custom policy Latest tag can be added and Violation is triggered: C811"() {
        String indexIP = System.getenv("index")
        List<String>violations = new ArrayList<>()
        String deployName = "qaport22"

        when: "Pull image Port 22"
        orchestrator.setDeploymentName(deployName)
        orchestrator.addContainerPort(22)
        assert orchestrator.createDeployment()

        and: "Policy is available to trigger the violations"
        Gson gson = new GsonBuilder().create()
        def res = getPolicies(indexIP)
        PolicyResults json_result_rule = gson.fromJson(res, PolicyResults)
        assert json_result_rule.policies.name.contains("Container Port 22")

        then: "Verify Violations have been triggered"
        def result = getViolations(indexIP)
        AlertsByPolicy json_result_violations = gson.fromJson(result, AlertsByPolicy)
        for(int i=0 ;i<json_result_violations.alertsByPolicies.size();i++){
            violations.add(json_result_violations.alertsByPolicies[i].policy.name)
        }
        assert violations.contains("Container Port 22")

        cleanup: "Remove Deployment"
        def teststatus = tc.verifyAndAdd(violations,"Container Port 22",811)
        resultMap.put(811, teststatus)
        orchestrator.deleteDeployment(deployName)

    }

}
