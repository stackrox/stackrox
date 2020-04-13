package services

import io.stackrox.proto.api.v1.MetadataServiceGrpc

class MetadataService extends BaseService {
    static getMetadataServiceClient() {
        return MetadataServiceGrpc.newBlockingStub(getChannel())
    }

    static boolean isReleaseBuild() {
        return getMetadataServiceClient().getMetadata().releaseBuild
    }
}
