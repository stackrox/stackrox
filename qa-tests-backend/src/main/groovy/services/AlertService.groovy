package services

import groovy.transform.CompileStatic

import io.stackrox.annotations.Retry
import io.stackrox.proto.api.v1.AlertServiceGrpc
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertTimeseriesResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsGroupResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ResolveAlertRequest
import io.stackrox.proto.storage.AlertOuterClass.Alert
import io.stackrox.proto.storage.AlertOuterClass.ListAlert

@CompileStatic
class AlertService extends BaseService {
    static AlertServiceGrpc.AlertServiceBlockingStub getAlertClient() {
        return AlertServiceGrpc.newBlockingStub(getChannel())
    }

    @Retry
    static List<ListAlert> getViolations(ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().listAlerts(request).alertsList
    }

    @Retry
    static GetAlertsCountsResponse getAlertCounts(
            GetAlertsCountsRequest request = GetAlertsCountsRequest.newBuilder().build()) {
        return getAlertClient().getAlertsCounts(request)
    }

    @Retry
    static GetAlertsGroupResponse getAlertGroups(ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().getAlertsGroup(request)
    }

    @Retry
    static GetAlertTimeseriesResponse getAlertTimeseries(
            ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().getAlertTimeseries(request)
    }

    @Retry
    static Alert getViolation(String alertId) {
        return getAlertClient().getAlert(getResourceByID(alertId))
    }

    @Retry
    static resolveAlert(String alertID, boolean addToBaseline = false) {
        return getAlertClient().resolveAlert(
                ResolveAlertRequest.newBuilder().setId(alertID).setAddToBaseline(addToBaseline).build())
    }
}
