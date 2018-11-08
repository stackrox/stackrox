import groups.BAT
import org.junit.experimental.categories.Category
import services.SummaryService
import stackrox.generated.SummaryServiceOuterClass

class SummaryTest extends BaseSpecification {
    @Category([BAT])
    def "Verify TopNav counts for Nodes, Deployments, and Secrets"() {
        expect:
        "Counts API should match orchestrator details"
        SummaryServiceOuterClass.SummaryCountsResponse counts = SummaryService.getCounts()

        def deployments = orchestrator.getDeploymentCount() + orchestrator.getDaemonSetCount()
        if (orchestrator.isKubeProxyPresent()) {
            deployments++ // Add 1 to deployment count to match "kube-proxy" deployment in GKE
        }

        assert counts.numDeployments == deployments
        assert counts.numSecrets == orchestrator.getSecretCount()
        assert counts.numNodes == orchestrator.getNodeCount()
    }
}
