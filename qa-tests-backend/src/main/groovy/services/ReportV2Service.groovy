package services

import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v2.Common
import io.stackrox.proto.api.v2.ReportServiceGrpc
import io.stackrox.proto.api.v2.ReportServiceOuterClass

@Slf4j
class ReportV2Service extends BaseService {

    static ReportServiceGrpc.ReportServiceBlockingStub getClient() {
        return ReportServiceGrpc.newBlockingStub(getChannel())
    }

    static ReportServiceOuterClass.ReportConfiguration createReportConfig(
            ReportServiceOuterClass.ReportConfiguration config) {
        return getClient().postReportConfiguration(config)
    }

    static deleteReportConfig(String configId) {
        getClient().deleteReportConfiguration(
                Common.ResourceByID.newBuilder().setId(configId).build())
    }

    static ReportServiceOuterClass.RunReportResponse runReport(String configId,
            ReportServiceOuterClass.NotificationMethod method) {
        return getClient().runReport(
                ReportServiceOuterClass.RunReportRequest.newBuilder()
                        .setReportConfigId(configId)
                        .setReportNotificationMethod(method)
                        .build())
    }

    static ReportServiceOuterClass.ReportStatusResponse getReportStatus(String reportId) {
        return getClient().getReportStatus(
                Common.ResourceByID.newBuilder().setId(reportId).build())
    }
}
