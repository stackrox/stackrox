package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.NodeServiceGrpc
import io.stackrox.proto.api.v1.NodeServiceOuterClass

@CompileStatic
class NodeService extends BaseService {
    static NodeServiceGrpc.NodeServiceBlockingStub getNodeClient() {
        return NodeServiceGrpc.newBlockingStub(getChannel())
            .withMaxInboundMessageSize(3 * 4194304) // Three times the default size, needed for multi-zone clusters
    }

    static getNodes(String clusterId = ClusterService.getClusterId()) {
        return getNodeClient().listNodes(
                NodeServiceOuterClass.ListNodesRequest.newBuilder()
                        .setClusterId(clusterId).build()
        ).nodesList
    }
}
