package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.NodeServiceGrpc
import io.stackrox.proto.api.v1.NodeServiceOuterClass

@CompileStatic
class NodeService extends BaseService {
    static NodeServiceGrpc.NodeServiceBlockingStub getNodeClient() {
        return NodeServiceGrpc.newBlockingStub(getChannel())
    }

    static getNodes(String clusterId = ClusterService.getClusterId()) {
        return getNodeClient().listNodes(
                NodeServiceOuterClass.ListNodesRequest.newBuilder()
                        .setClusterId(clusterId).build()
        ).nodesList
    }
}
