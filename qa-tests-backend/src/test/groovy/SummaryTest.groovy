import groups.BAT
import orchestratormanager.OrchestratorTypes
import org.junit.experimental.categories.Category
import services.SummaryService
import stackrox.generated.SummaryServiceOuterClass

class SummaryTest extends BaseSpecification {

    @Category([BAT])
    def "Verify TopNav counts for Nodes, Deployments, and Secrets"() {
        expect:
        "Counts API should match orchestrator details"
        def deployments = orchestrator.getDeploymentCount() + orchestrator.getDaemonSetCount()
        if (orchestrator.isKubeProxyPresent()) {
            deployments++ // Add 1 to deployment count to match "kube-proxy" deployment in GKE
        }
        if (OrchestratorTypes.valueOf(System.getenv("CLUSTER")) == OrchestratorTypes.OPENSHIFT) {
            deployments += 3 // Add 3 to deployment count in OS to account for kube-system pods
        }

        SummaryServiceOuterClass.SummaryCountsResponse counts = SummaryService.getCounts()
        def start = System.currentTimeMillis()
        while (counts.numDeployments != deployments && (System.currentTimeMillis() - start) < 30000) {
            counts = SummaryService.getCounts()
        }

        assert counts.numDeployments == deployments
        assert counts.numSecrets == orchestrator.getSecretCount()
        assert counts.numNodes == orchestrator.getNodeCount()
    }
}
