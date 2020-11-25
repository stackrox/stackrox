import { useSelector } from 'react-redux';
import { createSelector } from 'reselect';
import { selectors } from 'reducers';

const selectFeatureFlags = createSelector(
    [selectors.getFeatureFlags],
    (featureFlags) => featureFlags
);

const useFeatureFlagEnabled = (selectedFeatureFlag) => {
    const featureFlags = useSelector(selectFeatureFlags);
    const foundFeatureFlag = featureFlags.find(
        (featureFlag) => featureFlag.envVar === selectedFeatureFlag
    );

    if (foundFeatureFlag === null || foundFeatureFlag === undefined) {
        throw new Error(`Feature Flag (${selectedFeatureFlag}) is not a valid feature flag`);
    }
    return !!foundFeatureFlag.enabled;
};

export default useFeatureFlagEnabled;
