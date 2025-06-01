package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.NetworkBaselineServiceGrpc
import io.stackrox.proto.api.v1.NetworkBaselineServiceOuterClass
import io.stackrox.proto.api.v1.NetworkBaselineServiceOuterClass.NetworkBaselineExternalStatusRequest

@CompileStatic
class NetworkBaselineService extends BaseService {

    static NetworkBaselineServiceGrpc.NetworkBaselineServiceBlockingStub getNetworkBaselineClient() {
        return NetworkBaselineServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkBaseline(String deploymentID) {
        return getNetworkBaselineClient().getNetworkBaseline(ResourceByID.newBuilder().setId(deploymentID).build())
    }

    static getNetworkBaselineForExternalFlows(String deploymentID) {
        NetworkBaselineExternalStatusRequest.Builder request = NetworkBaselineExternalStatusRequest.newBuilder()
                                                                        .setDeploymentId(deploymentID)
        return getNetworkBaselineClient().getNetworkBaselineStatusForExternalFlows(request.build())
    }

    static lockNetworkBaseline(String deploymentID) {
        getNetworkBaselineClient().lockNetworkBaseline(ResourceByID.newBuilder().setId(deploymentID).build())
    }

    static modifyBaselineStatusForPeers(String deploymentID, NetworkBaselineServiceOuterClass.NetworkBaselinePeerStatus peer) {
        NetworkBaselineServiceOuterClass.ModifyBaselineStatusForPeersRequest request = NetworkBaselineServiceOuterClass.ModifyBaselineStatusForPeersRequest.newBuilder()
                                                                        .setDeploymentId(deploymentID)
                                                                        .addPeers(peer)
                                                                        .build()

        return getNetworkBaselineClient().modifyBaselineStatusForPeers(request)
    }
}
