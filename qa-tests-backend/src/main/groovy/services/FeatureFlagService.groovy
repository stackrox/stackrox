package services

import io.stackrox.proto.api.v1.FeatureFlagServiceGrpc
import util.E2ETestException

class FeatureFlagService extends BaseService {
    static getFeatureFlagServiceClient() {
        return FeatureFlagServiceGrpc.newBlockingStub(getChannel())
    }

    static getFeatureFlags() {
        return getFeatureFlagServiceClient().getFeatureFlags().featureFlagsList
    }

    static boolean isFeatureFlagEnabled(String envVar) {
        def flag = getFeatureFlags().find { it.envVar == envVar }
        if (!flag) {
            throw new E2ETestException("Could not find ${envVar}, maybe the feature flag was removed?")
        }
        return flag.enabled
    }
}
