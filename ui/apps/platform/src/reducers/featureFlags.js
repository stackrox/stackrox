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

const error = (state = null, action) => {
    switch (action.type) {
        case types.FETCH_FEATURE_FLAGS.REQUEST:
        case types.FETCH_FEATURE_FLAGS.SUCCESS:
            return null;

        case types.FETCH_FEATURE_FLAGS.FAILURE:
            return action.error;

        default:
            return state;
    }
};

const isLoading = (state = true, action) => {
    // Initialize true for edge case before featureFlagSagas call fetchFeatureFlags.
    switch (action.type) {
        case types.FETCH_FEATURE_FLAGS.REQUEST:
            return true;

        case types.FETCH_FEATURE_FLAGS.FAILURE:
        case types.FETCH_FEATURE_FLAGS.SUCCESS:
            return false;

        default:
            return state;
    }
};

const reducer = combineReducers({
    featureFlags,
    error,
    isLoading,
});

const getFeatureFlags = (state) => state.featureFlags;
const getFeatureFlagsError = (state) => state.error;
const getIsLoadingFeatureFlags = (state) => state.isLoading;

export const selectors = {
    getFeatureFlags,
    getFeatureFlagsError,
    getIsLoadingFeatureFlags,
};

export default reducer;
