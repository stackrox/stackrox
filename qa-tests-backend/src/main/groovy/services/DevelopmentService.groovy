package services

import io.stackrox.proto.internalapi.central.DevelopmentServiceGrpc
import io.stackrox.proto.internalapi.central.DevelopmentServiceOuterClass

class DevelopmentService extends BaseService {
    static getDevelopmentServiceClient() {
        return DevelopmentServiceGrpc.newBlockingStub(getChannel())
    }

    static DevelopmentServiceOuterClass.ReconciliationStatsByClusterResponse getReconciliationStatsByCluster() {
        return getDevelopmentServiceClient().reconciliationStatsByCluster()
    }
}
