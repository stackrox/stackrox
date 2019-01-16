package services

import io.stackrox.proto.api.v1.BenchmarkScanServiceGrpc
import io.stackrox.proto.api.v1.BenchmarkScanServiceOuterClass.ListBenchmarkScansRequest
import io.stackrox.proto.api.v1.BenchmarkScanServiceOuterClass.GetBenchmarkScanRequest
import io.stackrox.proto.api.v1.BenchmarkServiceGrpc
import io.stackrox.proto.api.v1.BenchmarkServiceOuterClass.GetBenchmarksRequest
import io.stackrox.proto.api.v1.BenchmarkTriggerServiceGrpc
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.BenchmarkScanOuterClass.BenchmarkScan
import io.stackrox.proto.storage.BenchmarkTriggerOuterClass.BenchmarkTrigger
import io.stackrox.proto.storage.Compliance
import io.stackrox.proto.storage.Compliance.ComplianceControlResult
import v1.ComplianceServiceGrpc
import v1.ComplianceServiceOuterClass.ComplianceStandard
import v1.ComplianceServiceOuterClass.ComplianceStandardMetadata

import java.util.stream.Collectors

class ComplianceService extends BaseService {
    private static final int MAX_WAIT_TIME = 45000
    private static final int SLEEP_DURATION = 3000

    static getComplianceClient() {
        return ComplianceServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ComplianceStandardMetadata> getComplianceStandards() {
        return getComplianceClient().getStandards().standardsList
    }

    static ComplianceStandard getComplianceStandardDetails(String complianceId) {
        try {
            return getComplianceClient()
                    .getStandard(Common.ResourceByID.newBuilder()
                    .setId(complianceId).build()).standard
        } catch (Exception e) {
            println "Could not find Compliance Standard with ID ${complianceId}: ${e.toString()}"
        }
    }

    static List<ComplianceControlResult> getComplianceResults(RawQuery query = RawQuery.newBuilder().build()) {
        // return getComplianceClient().getComplianceControlResults(query).resultsList
        // the results api is not sending any data yet... send mock data instead for testing purposes
        println query
        return [
                ComplianceControlResult.newBuilder()
                        .setControlId("fake-1.1.1")
                        .setValue(Compliance.ComplianceResultValue.newBuilder()
                                .addEvidence(
                                        Compliance.ComplianceResultValue.Evidence.newBuilder()
                                                .setState(Compliance.ComplianceState.COMPLIANCE_STATE_SUCCESS)
                                                .setMessage("Your deployments are frobotzed")
                                                .build()
                                ).build()
                        ).build(),
                ComplianceControlResult.newBuilder()
                        .setControlId("fake-1.1.2")
                        .setValue(Compliance.ComplianceResultValue.newBuilder()
                                .addEvidence(
                                        Compliance.ComplianceResultValue.Evidence.newBuilder()
                                                .setState(Compliance.ComplianceState.COMPLIANCE_STATE_FAILURE)
                                                .setMessage("Your nodes are not frobotzed")
                                                .build()
                                ).build()
                        ).build(),
        ]
    }

    /*
      Legacy Benchmark APIs
     */

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
