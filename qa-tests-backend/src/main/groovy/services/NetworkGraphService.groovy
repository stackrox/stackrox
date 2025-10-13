package services

import com.google.protobuf.Timestamp
import groovy.transform.CompileStatic
import groovy.transform.NullCheck
import groovy.util.logging.Slf4j

import io.stackrox.annotations.Retry
import io.stackrox.proto.api.v1.NetworkGraphServiceGrpc
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.CreateNetworkEntityRequest
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.GetExternalNetworkEntitiesRequest
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.GetExternalNetworkEntitiesResponse
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.NetworkGraphRequest
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass.NetworkGraphScope
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntity
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntityInfo.ExternalSource

@Slf4j
@CompileStatic
class NetworkGraphService extends BaseService {
    static NetworkGraphServiceGrpc.NetworkGraphServiceBlockingStub getNetworkGraphClient() {
        return NetworkGraphServiceGrpc.newBlockingStub(getChannel())
                .withMaxInboundMessageSize(2 * 4194304) // Twice the default size
                .withMaxOutboundMessageSize(2 * 4194304)
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

    @NullCheck
    static NetworkEntity createNetworkEntity(String clusterId, String name, String cidr, boolean isSystemGenerated) {
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

        return createNetworkEntity(request)
    }

    @Retry
    static NetworkEntity createNetworkEntity(CreateNetworkEntityRequest request) {
        getNetworkGraphClient().createExternalNetworkEntity(request)
    }

    @Retry(attempts = 24, delay = 5)
    static NetworkEntity waitForNetworkEntityOfExternalSource(String clusterId, String entityName) {
        GetExternalNetworkEntitiesRequest request =
                GetExternalNetworkEntitiesRequest.newBuilder().setClusterId(clusterId).build()
        GetExternalNetworkEntitiesResponse response =
                getNetworkGraphClient().getExternalNetworkEntities(request)

        // Calling response.getEntitiesList() may cause io.grpc.StatusRuntimeException: RESOURCE_EXHAUSTED
        return response
                .getEntitiesList()
                .find {
                    NetworkEntity it ->
                        it.getInfo().hasExternalSource() &&
                                it.getInfo().getExternalSource().name == entityName
                }
    }
}
