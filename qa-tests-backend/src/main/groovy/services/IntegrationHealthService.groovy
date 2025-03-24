package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.IntegrationHealthServiceGrpc

@CompileStatic
class IntegrationHealthService extends BaseService {
    static IntegrationHealthServiceGrpc.IntegrationHealthServiceBlockingStub getIntegrationHealthClient() {
        return IntegrationHealthServiceGrpc.newBlockingStub(getChannel())
    }

    static getVulnDefinitionsInfo() {
        return getIntegrationHealthClient().getVulnDefinitionsInfo(null)
    }

    static getDeclarativeConfigHealthInfo() {
        return getIntegrationHealthClient().getDeclarativeConfigs(null)
    }
}
