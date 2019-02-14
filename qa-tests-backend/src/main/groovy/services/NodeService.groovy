package services

import io.stackrox.proto.api.v1.NodeServiceGrpc
import io.stackrox.proto.api.v1.NodeServiceOuterClass

class NodeService extends BaseService {
    static getNodeClient() {
        return NodeServiceGrpc.newBlockingStub(getChannel())
    }

    static getNodes(String clusterId = ClusterService.getClusterId()) {
        return getNodeClient().listNodes(
                NodeServiceOuterClass.ListNodesRequest.newBuilder()
                        .setClusterId(clusterId).build()
        ).nodesList
    }
}
