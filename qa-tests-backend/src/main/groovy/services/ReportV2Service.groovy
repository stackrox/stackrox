package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v2.ReportServiceGrpc
import io.stackrox.proto.api.v2.ReportServiceOuterClass

@Slf4j
@CompileStatic
class ReportV2Service extends BaseService {

    static ReportServiceGrpc.ReportServiceBlockingStub getClient() {
        return ReportServiceGrpc.newBlockingStub(getChannel())
    }

    static ReportServiceOuterClass.ReportConfiguration createReportConfig(
            ReportServiceOuterClass.ReportConfiguration config) {
        return getClient().postReportConfiguration(config).getReportConfig()
    }

    static deleteReportConfig(String configId) {
        getClient().deleteReportConfiguration(
                ReportServiceOuterClass.ResourceByID.newBuilder().setId(configId).build())
    }

    static ReportServiceOuterClass.RunReportResponse runReport(String configId,
            ReportServiceOuterClass.ReportStatus.NotificationMethod method) {
        return getClient().runReport(
                ReportServiceOuterClass.RunReportRequest.newBuilder()
                        .setReportConfigId(configId)
                        .setReportNotificationMethod(method)
                        .build())
    }

    static ReportServiceOuterClass.ReportStatusResponse getReportStatus(String reportId) {
        return getClient().getReportStatus(
                ReportServiceOuterClass.ResourceByID.newBuilder().setId(reportId).build())
    }
}
