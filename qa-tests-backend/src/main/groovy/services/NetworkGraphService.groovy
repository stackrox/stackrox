package services

import com.google.protobuf.Timestamp
import stackrox.generated.NetworkGraphOuterClass
import stackrox.generated.NetworkGraphServiceGrpc

class NetworkGraphService extends BaseService {
    static getNetworkGraphClient() {
        return NetworkGraphServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkGraph(Timestamp since = null) {
        try {
            NetworkGraphOuterClass.NetworkGraphRequest.Builder request =
                    NetworkGraphOuterClass.NetworkGraphRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
            if (since != null) {
                request.setSince(since)
            }
            return getNetworkGraphClient().getNetworkGraph(request.build())
        } catch (Exception e) {
            println "Exception fetching network graph: ${e.toString()}"
        }
    }
}
