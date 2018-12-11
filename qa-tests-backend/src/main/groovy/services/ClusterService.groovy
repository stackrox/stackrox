package services

import io.stackrox.proto.api.v1.ClustersServiceGrpc
import io.stackrox.proto.api.v1.Common

class ClusterService extends BaseService {
    static getClusterServiceClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static getClusterId(String name = "remote") {
        return getClusterServiceClient().getClusters().clustersList.find { it.name == name }?.id
    }

    static createCluster(String name, String mainImage, String centralEndpoint) {
        return getClusterServiceClient().postCluster(io.stackrox.proto.api.v1.ClusterService.Cluster.newBuilder()
                .setName(name)
                .setMainImage(mainImage)
                .setCentralApiEndpoint(centralEndpoint)
                .build()
        )
    }

    static deleteCluster(String clusterId) {
        try {
            getClusterServiceClient().deleteCluster(Common.ResourceByID.newBuilder().setId(clusterId).build())
        } catch (Exception e) {
            println "Error deleting cluster: ${e}"
        }
    }
}
