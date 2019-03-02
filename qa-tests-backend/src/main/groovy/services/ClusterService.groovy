package services

import io.stackrox.proto.api.v1.ClustersServiceGrpc
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.storage.ClusterOuterClass.Cluster

class ClusterService extends BaseService {
    static getClusterServiceClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static List<Cluster> getClusters() {
        return getClusterServiceClient().getClusters().clustersList
    }

    static getClusterId(String name = "remote") {
        return getClusterServiceClient().getClusters().clustersList.find { it.name == name }?.id
    }

    static createCluster(String name, String mainImage, String centralEndpoint) {
        return getClusterServiceClient().postCluster(Cluster.newBuilder()
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
