package services

import com.google.protobuf.Timestamp
import io.stackrox.proto.api.v1.NetworkGraphOuterClass.NetworkGraphRequest
import io.stackrox.proto.api.v1.NetworkGraphServiceGrpc

class NetworkGraphService extends BaseService {
    static getNetworkGraphClient() {
        return NetworkGraphServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkGraph(Timestamp since = null, String query = null) {
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
            return getNetworkGraphClient().getNetworkGraph(request.build())
        } catch (Exception e) {
            println "Exception fetching network graph: ${e.toString()}"
        }
    }
}
