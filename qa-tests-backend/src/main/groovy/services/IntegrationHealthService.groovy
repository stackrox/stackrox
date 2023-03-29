package services

import io.stackrox.proto.api.v1.IntegrationHealthServiceGrpc

class IntegrationHealthService extends BaseService {
    static getIntegrationHealthClient() {
        return IntegrationHealthServiceGrpc.newBlockingStub(getChannel())
    }

    static getVulnDefinitionsInfo() {
        return getIntegrationHealthClient().getVulnDefinitionsInfo()
    }

    static getDeclarativeConfigHealthInfo() {
        return getIntegrationHealthClient().getDeclarativeConfigs()
    }
}
