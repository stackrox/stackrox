package services

import com.google.protobuf.Timestamp
import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ComplianceManagementServiceGrpc
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.ComplianceRunSelection
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.GetComplianceRunStatusesRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.GetRecentComplianceRunsRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.TriggerComplianceRunsRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.ComplianceRun

@Slf4j
class ComplianceManagementService extends BaseService {
    static getComplianceManagementClient() {
        return ComplianceManagementServiceGrpc.newBlockingStub(getChannel())
    }

    static triggerComplianceRuns(String standardId = null, String clusterId = null) {
        ComplianceRunSelection selection =
                ComplianceRunSelection.newBuilder()
                    .setStandardId(standardId ?: "*")
                    .setClusterId(clusterId ?: "*")
                    .build()
        try {
            return getComplianceManagementClient().triggerRuns(
                    TriggerComplianceRunsRequest.newBuilder().setSelection(selection).build()
            ).startedRunsList
        } catch (Exception e) {
            log.error("Error triggering compliance runs", e)
        }
    }

    static Map<String, String> triggerComplianceRunsAndWait(String standardId = null, String clusterId = null) {
        List<ComplianceRun> complianceRuns = triggerComplianceRuns(standardId, clusterId)
        log.debug "triggered ${standardId ?: "all"} compliance run${standardId ? "" : "s"}"
        log.debug "waiting for the run${standardId ? "" : "s"} to finish..."
        Long startTime = System.currentTimeMillis()
        while (complianceRuns.any { it.state != ComplianceRun.State.FINISHED } &&
                (System.currentTimeMillis() - startTime) < 300000) {
            sleep 1000
            complianceRuns = getRunStatuses(complianceRuns*.id).runsList
        }
        assert !complianceRuns.any { it.state != ComplianceRun.State.FINISHED }
        log.debug "Compliance run${standardId ? "" : "s"} took ${(System.currentTimeMillis() - startTime) / 1000}s"
        return complianceRuns.collectEntries { [(it.standardId) : it.id] }
    }

    static getRunStatuses(List<String> runIds) {
        return getComplianceManagementClient().getRunStatuses(
                GetComplianceRunStatusesRequest.newBuilder()
                        .addAllRunIds(runIds)
                        .build()
        )
    }

    static getRecentRuns(String standardId = null, String clusterId = null, Timestamp since = null) {
        GetRecentComplianceRunsRequest.Builder builder = GetRecentComplianceRunsRequest.newBuilder()
        standardId == null ?: builder.setStandardId(standardId)
        clusterId == null ?: builder.setClusterId(clusterId)
        since == null ?: builder.setSince(since)
        return getComplianceManagementClient().getRecentRuns(builder.build()).complianceRunsList
    }
}
