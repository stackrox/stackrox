import { combineReducers } from 'redux';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_FEATURE_FLAGS: createFetchingActionTypes('featureflags/FETCH_FEATURE_FLAGS'),
};

export const actions = {
    fetchFeatureFlags: createFetchingActions(types.FETCH_FEATURE_FLAGS),
};

const featureFlags = (state = [], action) => {
    if (action.type === types.FETCH_FEATURE_FLAGS.SUCCESS) {
        return action.response.featureFlags;
    }
    return state;
};

const featureFlagsError = (state = null, action) => {
    switch (action.type) {
        case types.FETCH_FEATURE_FLAGS.FAILURE:
            return action.error;

        case types.FETCH_FEATURE_FLAGS.SUCCESS:
            return null;

        default:
            return state;
    }
};

const isLoadingFeatureFlags = (state = true, action) => {
    // Assume featureFlagSagas call fetchFeatureFlags.
    switch (action.type) {
        case types.FETCH_FEATURE_FLAGS.FAILURE:
        case types.FETCH_FEATURE_FLAGS.SUCCESS:
            return false;

        default:
            return state;
    }
};

const reducer = combineReducers({
    featureFlags,
    featureFlagsError,
    isLoadingFeatureFlags,
});

const getFeatureFlags = (state) => state.featureFlags;
const getFeatureFlagsError = (state) => state.featureFlagsError;
const getIsLoadingFeatureFlags = (state) => state.isLoadingFeatureFlags;

export const selectors = {
    getFeatureFlags,
    getFeatureFlagsError,
    getIsLoadingFeatureFlags,
};

export default reducer;
