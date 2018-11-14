package services

import stackrox.generated.BenchmarkScanServiceGrpc
import stackrox.generated.BenchmarkScanServiceOuterClass.BenchmarkScan
import stackrox.generated.BenchmarkScanServiceOuterClass.ListBenchmarkScansRequest
import stackrox.generated.BenchmarkScanServiceOuterClass.GetBenchmarkScanRequest

import stackrox.generated.BenchmarkServiceGrpc
import stackrox.generated.BenchmarkServiceOuterClass.GetBenchmarksRequest
import stackrox.generated.BenchmarkTriggerServiceGrpc
import stackrox.generated.BenchmarkTriggerServiceOuterClass.BenchmarkTrigger

import java.util.stream.Collectors

class ComplianceService extends BaseService {
    private static final int MAX_WAIT_TIME = 45000
    private static final int SLEEP_DURATION = 3000

    static getBenchmarkTriggersClient() {
        return BenchmarkTriggerServiceGrpc.newBlockingStub(getChannel())
    }

    static getBenchmarkScansClient() {
        return BenchmarkScanServiceGrpc.newBlockingStub(getChannel())
    }

    static getBenchmarkClient() {
        return BenchmarkServiceGrpc.newBlockingStub(getChannel())
    }

    static getBenchmark(String name) {
        def benchmarkClient = getBenchmarkClient()
        def response = benchmarkClient.getBenchmarks(
                GetBenchmarksRequest.newBuilder().build())
        return response.getBenchmarksList().stream()
                .filter { f -> f.getName() == name }
                .collect(Collectors.toList())[0].getId()
    }

    static runBenchmark(String benchmarkID, String clusterID) {
        println "Running benchmark ${benchmarkID} on cluster ${clusterID}"
        def benchmarkTriggerClient = getBenchmarkTriggersClient()
        benchmarkTriggerClient.trigger(
                BenchmarkTrigger.newBuilder().
                        addClusterIds(clusterID).
                        setId(benchmarkID).build()
        )
    }

    static checkBenchmarkRan(String benchmarkID, String clusterID) {
        def scanClient = getBenchmarkScansClient()

        int waitTime = 0
        while (waitTime < MAX_WAIT_TIME) {
            def scanResponse = scanClient.listBenchmarkScans(
                    ListBenchmarkScansRequest.newBuilder().
                            setBenchmarkId(benchmarkID).
                            addClusterIds(clusterID).build()
            )
            if (scanResponse.getScanMetadataList().size() != 0) {
                def scanID = scanResponse.getScanMetadataList()[0].getScanId()
                try {
                    BenchmarkScan scan = scanClient.getBenchmarkScan(
                            GetBenchmarkScanRequest.newBuilder().setScanId(scanID).build())
                    if (scan.getChecksList().get(0).getAggregatedResultsCount() > 0) {
                        return true
                    }
                    println "Got scan, but no benchmarks had reported back"
                } catch (Exception e) {
                    println "Unable to get benchmark scan: ${e.toString()}"
                }
            }
            sleep(SLEEP_DURATION)
            waitTime += SLEEP_DURATION
        }
        return false
    }

}
