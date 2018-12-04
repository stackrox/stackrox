package services

import io.stackrox.proto.api.v1.ClustersServiceGrpc

class ClusterService extends BaseService {
    static getClusterServiceClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static getClusterId(String name = "remote") {
        return getClusterServiceClient().getClusters().clustersList.find { it.name == name }?.id
    }
}
