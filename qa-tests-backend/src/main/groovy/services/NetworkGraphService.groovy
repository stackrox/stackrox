package services

import com.google.protobuf.Timestamp
import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.CreateNetworkEntityRequest
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.GetExternalNetworkEntitiesRequest
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.GetExternalNetworkEntitiesResponse
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.NetworkGraphRequest
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.NetworkGraphScope
import io.stackrox.proto.api.v1.NetworkGraphServiceGrpc
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntity
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntityInfo.ExternalSource
import util.Timer

@Slf4j
class NetworkGraphService extends BaseService {
    static getNetworkGraphClient() {
        return NetworkGraphServiceGrpc.newBlockingStub(getChannel())
            .withMaxInboundMessageSize(2*4209569)
            .withMaxOutboundMessageSize(2*4209569)
    }

    static getNetworkGraph(Timestamp since = null, String query = null, String scopeQuery = null) {
        try {
            NetworkGraphRequest.Builder request =
                    NetworkGraphRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
            if (since != null) {
                request.setSince(since)
            }
            if (query != null) {
                request.setQuery(query)
            }
            if (scopeQuery != null) {
                request.setScope(NetworkGraphScope.newBuilder().setQuery(scopeQuery))
            }
            return getNetworkGraphClient().getNetworkGraph(request.build())
        } catch (Exception e) {
            log.error("Exception fetching network graph", e)
        }
    }

    static createNetworkEntity(String clusterId, String name, String cidr, Boolean isSystemGenerated) {
        try {
            if (clusterId == null) {
                throw new RuntimeException("Cluster ID is required to create a network entity")
            }
            if (name == null) {
                throw new RuntimeException("Name is required to create a network entity")
            }
            if (cidr == null) {
                throw new RuntimeException("CIDR address needs to be defined to create a network entity")
            }
            // Create entity for request
            ExternalSource.Builder entity =
                    ExternalSource
                            .newBuilder()
                            .setName(name)
                            .setCidr(cidr)
                            .setDefault(isSystemGenerated)

            // Create request
            CreateNetworkEntityRequest request =
                    CreateNetworkEntityRequest
                            .newBuilder()
                            .setClusterId(clusterId)
                            .setEntity(entity)
                            .build()

            return getNetworkGraphClient().createExternalNetworkEntity(request)
        } catch (Exception e) {
            log.error("Exception while creating network entity", e)
        }
    }

    static deleteNetworkEntity(String entityID) {
        try {
            // Create request
            getNetworkGraphClient().deleteExternalNetworkEntity(ResourceByID.newBuilder().setId(entityID).build())
        } catch (Exception e) {
            log.error("Exception while deleting network entity", e)
        }
    }

    static waitForNetworkEntityOfExternalSource(String clusterId, String entityName) {
        int intervalInSeconds = 5
        int timeoutInSeconds = 120
        int retries = timeoutInSeconds / intervalInSeconds
        Timer t = new Timer(retries, intervalInSeconds)
        while (t.IsValid()) {
            try {
                GetExternalNetworkEntitiesRequest request =
                        GetExternalNetworkEntitiesRequest.newBuilder().setClusterId(clusterId).build()
                GetExternalNetworkEntitiesResponse response =
                        getNetworkGraphClient().getExternalNetworkEntities(request)
                NetworkEntity matchingEntity =
                        response
                                .getEntitiesList()
                                .find {
                                    NetworkEntity it -> it.getInfo().hasExternalSource() &&
                                            it.getInfo().getExternalSource().name == entityName
                                }
                if (matchingEntity != null) {
                    return matchingEntity
                }
            } catch (Exception e) {
                log.debug("Exception while getting network entity with name ${entityName}, retrying...", e)
            }
        }
        log.warn "Failed to get network entity with name ${entityName} under cluster ${clusterId}"
    }
}
