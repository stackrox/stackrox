package services

import groovy.transform.CompileStatic

import io.stackrox.proto.internalapi.central.DevelopmentServiceGrpc
import io.stackrox.proto.internalapi.central.DevelopmentServiceOuterClass

@CompileStatic
class DevelopmentService extends BaseService {
    static DevelopmentServiceGrpc.DevelopmentServiceBlockingStub getDevelopmentServiceClient() {
        return DevelopmentServiceGrpc.newBlockingStub(getChannel())
    }

    static DevelopmentServiceOuterClass.ReconciliationStatsByClusterResponse getReconciliationStatsByCluster() {
        return getDevelopmentServiceClient().reconciliationStatsByCluster(null)
    }
}
