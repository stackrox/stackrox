package services

import io.stackrox.proto.api.v1.AlertServiceGrpc
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsGroupResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertTimeseriesResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ResolveAlertRequest
import io.stackrox.proto.storage.AlertOuterClass.Alert
import io.stackrox.proto.storage.AlertOuterClass.ListAlert

class AlertService extends BaseService {
    static getAlertClient() {
        return AlertServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListAlert> getViolations(ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().listAlerts(request).alertsList
    }

    static GetAlertsCountsResponse getAlertCounts(
            GetAlertsCountsRequest request = GetAlertsCountsRequest.newBuilder().build()) {
        return getAlertClient().getAlertsCounts(request)
    }

    static GetAlertsGroupResponse getAlertGroups(ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().getAlertsGroup(request)
    }

    static GetAlertTimeseriesResponse getAlertTimeseries(
            ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().getAlertTimeseries(request)
    }

    static Alert getViolation(String alertId) {
        return getAlertClient().getAlert(getResourceByID(alertId))
    }

    static resolveAlert(String alertID) {
        return getAlertClient().resolveAlert(
                ResolveAlertRequest.newBuilder().setId(alertID).build())
    }
}
