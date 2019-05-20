package services

import io.stackrox.proto.api.v1.ClustersServiceGrpc
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.ClusterOuterClass.Cluster
import io.stackrox.proto.storage.ClusterOuterClass.DynamicClusterConfig

import java.util.stream.Collectors

class ClusterService extends BaseService {
    static getClusterServiceClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static List<Cluster> getClusters() {
        return getClusterServiceClient().getClusters().clustersList
    }

    static Cluster getCluster() {
        String clusterId = getClusterId()
        return getClusters().stream().filter { x -> x.id == clusterId }.collect(Collectors.toList()).first()
    }

    static getClusterId(String name = "remote") {
        try {
            return getClusterServiceClient().getClusters().clustersList.find { it.name == name }?.id
        } catch (Exception e) {
            println "Error getting cluster ID: ${e}"
            return e
        }
    }

    static createCluster(String name, String mainImage, String centralEndpoint) {
        try {
            return getClusterServiceClient().postCluster(Cluster.newBuilder()
                    .setName(name)
                    .setMainImage(mainImage)
                    .setCentralApiEndpoint(centralEndpoint)
                    .build()
            )
        } catch (Exception e) {
            println "Error creating cluster: ${e}"
            return e
        }
    }

    static Boolean updateAdmissionController(AdmissionControllerConfig config) {
        Cluster currentCluster = getCluster()
        if (currentCluster == null) {
            return false
        }
        Cluster.Builder builder = currentCluster.toBuilder()

        Cluster cluster = builder.setDynamicConfig(
                DynamicClusterConfig.newBuilder()
                        .setAdmissionControllerConfig(config)
                        .build()
        ).build()

        return updateCluster(cluster)
    }

    static Boolean updateCluster(Cluster cluster)  {
        try {
            getClusterServiceClient().putCluster(cluster)
            return true
        } catch (Exception e) {
            println "Error creating cluster: ${e}"
            return false
        }
    }

    static deleteCluster(String clusterId) {
        try {
            getClusterServiceClient().deleteCluster(Common.ResourceByID.newBuilder().setId(clusterId).build())
        } catch (Exception e) {
            println "Error deleting cluster: ${e}"
        }
    }
}
