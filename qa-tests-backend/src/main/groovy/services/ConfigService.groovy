package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.ConfigServiceGrpc
import io.stackrox.proto.storage.ConfigOuterClass

@CompileStatic
class ConfigService extends BaseService {
    static ConfigServiceGrpc.ConfigServiceBlockingStub getConfigClient() {
        return ConfigServiceGrpc.newBlockingStub(getChannel())
    }

    static ConfigOuterClass.Config getConfig() {
        return getConfigClient().getConfig(EMPTY)
    }

    static getPublicConfig() {
        return getConfigClient().getPublicConfig(EMPTY)
    }

    static getPrivateConfig() {
        return getConfigClient().getPrivateConfig(EMPTY)
    }
}
