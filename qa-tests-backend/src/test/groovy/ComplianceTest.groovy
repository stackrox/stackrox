import services.ClusterService
import spock.lang.Unroll

import groups.BAT
import org.junit.experimental.categories.Category
import services.ComplianceService

class ComplianceTest extends BaseSpecification {
    @Unroll
    @Category(BAT)
    def "Verify that we can run a benchmark: "(String benchmarkName) {
        when:
        "Trigger a compliance benchmark"
        String benchmarkID = ComplianceService.getBenchmark(benchmarkName)
        println ("Found benchmark ID ${benchmarkID} for ${benchmarkName}")
        String clusterID = ClusterService.getClusterId()
        ComplianceService.runBenchmark(benchmarkID, clusterID)

        then:
        "Verify Scan is run"
        assert ComplianceService.checkBenchmarkRan(benchmarkID, clusterID)

        cleanup:
        "Make sure the daemonset benchmark is gone"
        orchestrator.waitForDaemonSetDeletion("benchmark", "stackrox")

        where:
        "Data inputs are :"

        benchmarkName | _
        "CIS Kubernetes v1.2.0 Benchmark" | _
        "CIS Docker v1.1.0 Benchmark" | _
    }

}
