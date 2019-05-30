package services

import io.stackrox.proto.api.v1.FeatureFlagServiceGrpc

class FeatureFlagService extends BaseService {
    static getFeatureFlagServiceClient() {
        return FeatureFlagServiceGrpc.newBlockingStub(getChannel())
    }

    static getFeatureFlags() {
        return getFeatureFlagServiceClient().getFeatureFlags().featureFlagsList
    }

    static isFeatureFlagEnabled(String envVar) {
        return getFeatureFlags().find { it.envVar == envVar }?.enabled
    }
}
