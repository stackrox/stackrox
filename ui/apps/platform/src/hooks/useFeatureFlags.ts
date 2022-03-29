import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { FeatureFlag } from 'types/featureFlagService.proto';

const featureFlagsSelector = createStructuredSelector<{ featureFlags: FeatureFlag[] }>({
    featureFlags: selectors.getFeatureFlags,
});

type UseFeatureFlags = {
    isFeatureFlagEnabled: (envVar: string) => boolean;
};

function useFeatureFlags(): UseFeatureFlags {
    const { featureFlags } = useSelector(featureFlagsSelector);

    function isFeatureFlagEnabled(envVar: string): boolean {
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

    return { isFeatureFlagEnabled };
}

export default useFeatureFlags;
