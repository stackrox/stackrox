package services

import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.NetworkBaselineServiceGrpc

class NetworkBaselineService extends BaseService {

    static getNetworkBaselineClient() {
        return NetworkBaselineServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkBaseline(String deploymentID) {
        return getNetworkBaselineClient().getNetworkBaseline(ResourceByID.newBuilder().setId(deploymentID).build())
    }
}
