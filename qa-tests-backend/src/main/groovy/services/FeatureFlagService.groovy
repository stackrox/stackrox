package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.FeatureFlagServiceGrpc
import io.stackrox.proto.api.v1.FeatureFlagServiceOuterClass

import util.E2ETestException

@CompileStatic
class FeatureFlagService extends BaseService {
    static FeatureFlagServiceGrpc.FeatureFlagServiceBlockingStub getFeatureFlagServiceClient() {
        return FeatureFlagServiceGrpc.newBlockingStub(getChannel())
    }

    static List<FeatureFlagServiceOuterClass.FeatureFlag> getFeatureFlags() {
        return getFeatureFlagServiceClient().getFeatureFlags(null).featureFlagsList
    }

    static boolean isFeatureFlagEnabled(String envVar) {
        def flag = getFeatureFlags().find { it.envVar == envVar }
        if (!flag) {
            throw new E2ETestException("Could not find ${envVar}, maybe the feature flag was removed?")
        }
        return flag.enabled
    }
}
