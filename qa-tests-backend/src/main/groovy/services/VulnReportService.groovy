package services

import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.ReportConfigurationServiceGrpc
import io.stackrox.proto.api.v1.ReportConfigurationServiceOuterClass
import io.stackrox.proto.api.v1.ReportServiceGrpc
import io.stackrox.proto.storage.Cve
import io.stackrox.proto.storage.ReportConfigurationOuterClass
import io.stackrox.proto.storage.ScheduleOuterClass

import common.Constants

@Slf4j
class VulnReportService extends BaseService {

    private static final ALL_SEVERITIES = [
            Cve.VulnerabilitySeverity.CRITICAL_VULNERABILITY_SEVERITY,
            Cve.VulnerabilitySeverity.IMPORTANT_VULNERABILITY_SEVERITY,
            Cve.VulnerabilitySeverity.MODERATE_VULNERABILITY_SEVERITY,
            Cve.VulnerabilitySeverity.LOW_VULNERABILITY_SEVERITY,
    ]

    static getConfigClient() {
        return ReportConfigurationServiceGrpc.newBlockingStub(getChannel())
    }

    static getReportSvcClient() {
        return ReportServiceGrpc.newBlockingStub(getChannel())
    }

    // Create a vulnerability report config whose scope is the specified collection
    // and uses the specified notifier to send the report
    static ReportConfigurationOuterClass.ReportConfiguration createVulnReportConfig(
            String collectionId,
            String notifierId) {
        def req = ReportConfigurationServiceOuterClass.PostReportConfigurationRequest.newBuilder()
        .setReportConfig(ReportConfigurationOuterClass.ReportConfiguration.newBuilder()
                .setName("Test Vuln Report-${UUID.randomUUID()}")
                .setType(ReportConfigurationOuterClass.ReportConfiguration.ReportType.VULNERABILITY)
                .setVulnReportFilters(ReportConfigurationOuterClass.VulnerabilityReportFilters.newBuilder()
                    .setFixability(ReportConfigurationOuterClass.VulnerabilityReportFilters.Fixability.BOTH)
                    .setSinceLastReport(false)
                    .addAllSeverities(ALL_SEVERITIES))
                .setScopeId(collectionId)
                .setEmailConfig(ReportConfigurationOuterClass.EmailNotifierConfiguration.newBuilder()
                    .setNotifierId(notifierId)
                    .addMailingLists(Constants.EMAIL_NOTIFER_SENDER))
                .setSchedule(ScheduleOuterClass.Schedule.newBuilder()
                    .setIntervalType(ScheduleOuterClass.Schedule.IntervalType.WEEKLY)
                    .setHour(0)
                    .setMinute(0)
                    .setDaysOfWeek(ScheduleOuterClass.Schedule.DaysOfWeek.newBuilder().addDays(1)))
        )
        return getConfigClient().postReportConfiguration(req.build()).getReportConfig()
    }

    static deleteVulnReportConfig(String reportId) {
        getConfigClient().deleteReportConfiguration(Common.ResourceByID.newBuilder().setId(reportId).build())
    }

    static boolean runReport(String reportConfigId) {
        try {
            getReportSvcClient().runReport(Common.ResourceByID.newBuilder().setId(reportConfigId).build())
        } catch (Exception e) {
            log.error("error running report with id " + reportConfigId, e)
            return false
        }
    }
}
