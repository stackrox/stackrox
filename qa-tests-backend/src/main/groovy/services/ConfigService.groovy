package services

import io.stackrox.proto.api.v1.ConfigServiceGrpc

class ConfigService extends BaseService {
    static getConfigClient() {
        return ConfigServiceGrpc.newBlockingStub(getChannel())
    }

    static getConfig() {
        return getConfigClient().getConfig()
    }

    static getPublicConfig() {
        return getConfigClient().getPublicConfig()
    }

    static getPrivateConfig() {
        return getConfigClient().getPrivateConfig()
    }
}
