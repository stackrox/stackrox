package services

import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.DiscoveredClusterService
import io.stackrox.proto.api.v1.DiscoveredClustersServiceGrpc

@CompileStatic
@Slf4j
class DiscoveredClustersService extends BaseService {
    static DiscoveredClustersServiceGrpc.DiscoveredClustersServiceBlockingStub getDiscoveredClustersClient() {
        return DiscoveredClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static Integer countDiscoveredClusters() {
        Integer count = -1
        try {
            count = getDiscoveredClustersClient().countDiscoveredClusters(DiscoveredClusterService.
                    CountDiscoveredClustersRequest.newBuilder().build()).getCount()
        } catch (Exception e) {
            log.error("Failed to count discovered clusters", e)
        }
        return count
    }
}
