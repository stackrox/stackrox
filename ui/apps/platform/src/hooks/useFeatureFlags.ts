import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { FeatureFlagEnvVar } from 'types/featureFlag';
import { FeatureFlag } from 'types/featureFlagService.proto';

const featureFlagsSelector = createStructuredSelector<{
    featureFlags: FeatureFlag[];
    isLoadingFeatureFlags: boolean;
}>({
    featureFlags: selectors.getFeatureFlags,
    isLoadingFeatureFlags: selectors.getIsLoadingFeatureFlags,
});

export type IsFeatureFlagEnabled = (envVar: FeatureFlagEnvVar) => boolean;

type UseFeatureFlagsResult = {
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
    isLoadingFeatureFlags: boolean;
};

function useFeatureFlags(): UseFeatureFlagsResult {
    const { featureFlags, isLoadingFeatureFlags } = useSelector(featureFlagsSelector);

    function isFeatureFlagEnabled(envVar: FeatureFlagEnvVar): boolean {
        const featureFlag = featureFlags.find((flag) => flag.envVar === envVar);
        if (!featureFlag) {
            if (process.env.NODE_ENV === 'development') {
                // eslint-disable-next-line no-console
                console.warn(`EnvVar ${envVar} not found in the backend list, possibly stale?`);
            }
            return false;
        }
        return featureFlag.enabled;
    }

    return { isFeatureFlagEnabled, isLoadingFeatureFlags };
}

export default useFeatureFlags;
