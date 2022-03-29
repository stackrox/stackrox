import { useSelector } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { FeatureFlag } from 'types/featureFlagService.proto';

const featureFlagsSelector = createStructuredSelector<{ featureFlags: FeatureFlag[] }>({
    featureFlags: selectors.getFeatureFlags,
});

function useFeatureFlags(): FeatureFlag[] {
    const { featureFlags } = useSelector(featureFlagsSelector);

    return featureFlags;
}

export default useFeatureFlags;
