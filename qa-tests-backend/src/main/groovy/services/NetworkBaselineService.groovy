package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.NetworkBaselineServiceGrpc

@CompileStatic
class NetworkBaselineService extends BaseService {

    static NetworkBaselineServiceGrpc.NetworkBaselineServiceBlockingStub getNetworkBaselineClient() {
        return NetworkBaselineServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkBaseline(String deploymentID) {
        return getNetworkBaselineClient().getNetworkBaseline(ResourceByID.newBuilder().setId(deploymentID).build())
    }

    static lockNetworkBaseline(String deploymentID) {
        getNetworkBaselineClient().lockNetworkBaseline(ResourceByID.newBuilder().setId(deploymentID).build())
    }
}
