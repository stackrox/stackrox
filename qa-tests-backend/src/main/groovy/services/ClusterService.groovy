package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.ClusterService.GetClustersRequest
import io.stackrox.proto.api.v1.ClustersServiceGrpc
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.storage.ClusterOuterClass
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.ClusterOuterClass.Cluster
import io.stackrox.proto.storage.ClusterOuterClass.DynamicClusterConfig

@CompileStatic
@Slf4j
class ClusterService extends BaseService {
    static final DEFAULT_CLUSTER_NAME = "remote"

    static ClustersServiceGrpc.ClustersServiceBlockingStub getClusterServiceClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static List<Cluster> getClusters() {
        return getClusterServiceClient().getClusters(null).clustersList
    }

    static Cluster getCluster() {
        String clusterId = getClusterId()
        return getClusterServiceClient().getCluster(Common.ResourceByID.newBuilder().setId(clusterId).build()).cluster
    }

    static getClusterId(String name = DEFAULT_CLUSTER_NAME) {
        return getClusterServiceClient().getClusters(
                GetClustersRequest.newBuilder().setQuery("Cluster:${name}").build()
        ).clustersList.find { it.name == name }?.id
    }

    static createCluster(String name, String mainImage, String centralEndpoint) {
        return getClusterServiceClient().postCluster(Cluster.newBuilder()
                .setName(name)
                .setMainImage(mainImage)
                .setCentralApiEndpoint(centralEndpoint)
                .build()
        )
    }

    static Boolean updateAdmissionController(AdmissionControllerConfig.Builder builder) {
        return updateAdmissionController(builder.build())
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

    static Boolean updateAuditLogDynamicConfig(boolean disableAuditLogs) {
        Cluster currentCluster = getCluster()
        if (currentCluster == null) {
            return false
        }
        Cluster.Builder builder = currentCluster.toBuilder()

        Cluster cluster = builder.setDynamicConfig(
                DynamicClusterConfig.newBuilder()
                        .setDisableAuditLogs(disableAuditLogs)
                        .build()
        ).build()

        return updateCluster(cluster)
    }

    static Boolean updateCluster(Cluster cluster) {
        try {
            getClusterServiceClient().putCluster(cluster)
            return true
        } catch (Exception e) {
            log.error("Error updating cluster", e)
            return false
        }
    }

    static deleteCluster(String clusterId) {
        try {
            getClusterServiceClient().deleteCluster(Common.ResourceByID.newBuilder().setId(clusterId).build())
        } catch (Exception e) {
            log.error("Error deleting cluster", e)
        }
    }

    static Boolean isEKS() {
        Boolean isEKS = false
        try {
            isEKS = clusters.every {
                Cluster cluster ->
                    cluster.getStatus().getProviderMetadata().hasAws() &&
                            cluster.getStatus().getOrchestratorMetadata().getVersion().contains("eks")
            }
        } catch (Exception e) {
            log.error("Error getting cluster info", e)
        }
        isEKS
    }

    static Boolean isAKS() {
        Boolean isAKS = false
        try {
            isAKS = clusters.every {
                Cluster cluster -> cluster.getStatus().getProviderMetadata().hasAzure()
            }
        } catch (Exception e) {
            log.error("Error getting cluster info", e)
        }
        isAKS
    }

    static Boolean isOpenShift3() {
        return getCluster().getType() == ClusterOuterClass.ClusterType.OPENSHIFT_CLUSTER
    }

    static Boolean isOpenShift4() {
        return getCluster().getType() == ClusterOuterClass.ClusterType.OPENSHIFT4_CLUSTER
    }
}
