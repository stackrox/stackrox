package services

import static io.stackrox.proto.api.v1.ClusterInitServiceOuterClass.InitBundleGenRequest
import static io.stackrox.proto.api.v1.ClusterInitServiceOuterClass.InitBundleGenResponse
import static io.stackrox.proto.api.v1.ClusterInitServiceOuterClass.InitBundleMeta
import static io.stackrox.proto.api.v1.ClusterInitServiceOuterClass.InitBundleRevokeRequest.newBuilder
import static io.stackrox.proto.api.v1.ClusterInitServiceOuterClass.InitBundleRevokeResponse

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.ClusterInitServiceGrpc

@CompileStatic
class ClusterInitBundleService extends BaseService {
    static ClusterInitServiceGrpc.ClusterInitServiceBlockingStub getClusterServiceClient() {
        return ClusterInitServiceGrpc.newBlockingStub(getChannel())
    }

    static List<InitBundleMeta> getInitBundles() {
        return getClusterServiceClient().getInitBundles(null)?.itemsList
    }

    static InitBundleGenResponse generateInintBundle(String name) {
        return getClusterServiceClient().generateInitBundle(InitBundleGenRequest.newBuilder().setName(name).build())
    }

    static InitBundleRevokeResponse revokeInitBundle(String bundleId) {
        return getClusterServiceClient().revokeInitBundle(newBuilder().addIds(bundleId).build())
    }
}
