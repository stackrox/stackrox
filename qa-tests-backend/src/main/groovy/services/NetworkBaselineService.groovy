package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.NetworkBaselineServiceGrpc
import io.stackrox.proto.api.v1.NetworkBaselineServiceOuterClass.NetworkBaselineExternalStatusRequest
import io.stackrox.proto.api.v1.NetworkBaselineServiceOuterClass.NetworkBaselinePeerStatus
import io.stackrox.proto.api.v1.NetworkBaselineServiceOuterClass.ModifyBaselineStatusForPeersRequest

@CompileStatic
class NetworkBaselineService extends BaseService {

    static NetworkBaselineServiceGrpc.NetworkBaselineServiceBlockingStub getNetworkBaselineClient() {
        return NetworkBaselineServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkBaseline(String deploymentID) {
        return getNetworkBaselineClient().getNetworkBaseline(ResourceByID.newBuilder().setId(deploymentID).build())
    }

    static getNetworkBaselineForExternalFlows(String deploymentID) {
        NetworkBaselineExternalStatusRequest request = NetworkBaselineExternalStatusRequest.newBuilder()
                                                                        .setDeploymentId(deploymentID)
                                                                        .build()

        return getNetworkBaselineClient().getNetworkBaselineStatusForExternalFlows(request)
    }

    static lockNetworkBaseline(String deploymentID) {
        getNetworkBaselineClient().lockNetworkBaseline(ResourceByID.newBuilder().setId(deploymentID).build())
    }

    static modifyBaselineStatusForPeers(String deploymentID, NetworkBaselinePeerStatus peer) {
        ModifyBaselineStatusForPeersRequest request = ModifyBaselineStatusForPeersRequest.newBuilder()
                                                                        .setDeploymentId(deploymentID)
                                                                        .addPeers(peer)
                                                                        .build()

        return getNetworkBaselineClient().modifyBaselineStatusForPeers(request)
    }
}
