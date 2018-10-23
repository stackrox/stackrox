package services

import stackrox.generated.ClustersServiceGrpc

class ClusterService extends BaseService {
    static getClusterServiceClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static getClusterId(String name = "remote") {
        return getClusterServiceClient().getClusters().clustersList.find { it.name == name }?.id
    }
}
