package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.MetadataServiceGrpc

@CompileStatic
class MetadataService extends BaseService {
    static MetadataServiceGrpc.MetadataServiceBlockingStub getMetadataServiceClient() {
        return MetadataServiceGrpc.newBlockingStub(getChannel())
    }

    static boolean isReleaseBuild() {
        return getMetadataServiceClient().getMetadata(null).releaseBuild
    }
}
