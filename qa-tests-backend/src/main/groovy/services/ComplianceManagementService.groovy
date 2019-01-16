package services

import com.google.protobuf.Timestamp
import io.stackrox.proto.api.v1.ComplianceManagementServiceGrpc
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.AddComplianceRunScheduleRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.DeleteComplianceRunScheduleRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.GetComplianceRunSchedulesRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.GetRecentComplianceRunsRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.TriggerComplianceRunRequest
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.UpdateComplianceRunScheduleRequest
import io.stackrox.proto.storage.ComplianceManagement.ComplianceRunSchedule

class ComplianceManagementService extends BaseService {
    static getComplianceManagementClient() {
        return ComplianceManagementServiceGrpc.newBlockingStub(getChannel())
    }

    static triggerComplianceRun(String standardId, String clusterId) {
        try {
            return getComplianceManagementClient().triggerRun(
                    TriggerComplianceRunRequest.newBuilder()
                            .setStandardId(standardId)
                            .setClusterId(clusterId)
                            .build()
            ).startedRun
        } catch (Exception e) {
            println "Error triggering compliance run: ${e.toString()}"
        }
    }

    static getRecentRuns(String standardId = null, String clusterId = null, Timestamp since = null) {
        GetRecentComplianceRunsRequest.Builder builder = GetRecentComplianceRunsRequest.newBuilder()
        standardId == null ?: builder.setStandardId(standardId)
        clusterId == null ?: builder.setClusterId(clusterId)
        since == null ?: builder.setSince(since)
        return getComplianceManagementClient().getRecentRuns(builder.build()).complianceRunsList
    }

    static getSchedules(String standardId = null, String clusterId = null, Boolean suspended = null) {
        GetComplianceRunSchedulesRequest.Builder builder = GetComplianceRunSchedulesRequest.newBuilder()
        standardId == null ?: builder.setStandardId(standardId)
        clusterId == null ?: builder.setClusterId(clusterId)
        suspended == null ?: builder.setSuspended(suspended)
        return getComplianceManagementClient().getRunSchedules(builder.build()).schedulesList
    }

    static addSchedule(String standardId, String clusterId, String crontab, Boolean suspended = false) {
        try {
            ComplianceRunSchedule.Builder builder = ComplianceRunSchedule.newBuilder()
                    .setStandardId(standardId)
                    .setClusterId(clusterId)
                    .setCrontabSpec(crontab)
                    .setSuspended(suspended)
            return getComplianceManagementClient().addRunSchedule(
                    AddComplianceRunScheduleRequest.newBuilder()
                            .setScheduleSpec(builder.build()
                    ).build()
            ).addedSchedule
        } catch (Exception e) {
            println "Error adding a compliance schedule: ${e.toString()}"
        }
    }

    static updateSchedule(
            String scheduleId,
            String standardId,
            String clusterId,
            String crontab = null,
            Boolean suspended = null) {
        try {
            ComplianceRunSchedule.Builder builder = ComplianceRunSchedule.newBuilder()
                    .setClusterId(clusterId)
                    .setStandardId(standardId)
            crontab == null ?: builder.setCrontabSpec(crontab)
            suspended == null ?: builder.setSuspended(suspended)
            return getComplianceManagementClient().updateRunSchedule(
                    UpdateComplianceRunScheduleRequest.newBuilder()
                            .setScheduleId(scheduleId)
                            .setUpdatedSpec(builder.build()
                    ).build()
            ).updatedSchedule
        } catch (Exception e) {
            println "Error updating a compliance schedule: ${e.toString()}"
        }
    }

    static deleteSchedule(String scheduleId) {
        try {
            getComplianceManagementClient().deleteRunSchedule(
                    DeleteComplianceRunScheduleRequest.newBuilder()
                            .setScheduleId(scheduleId)
                            .build()
            )
        } catch (Exception e) {
            println "Error deleting compliance schedule: ${e.toString()}"
        }
    }
}
