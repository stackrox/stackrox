import { combineReducers } from 'redux';

import { createFetchingActionTypes } from 'utils/fetchingReduxRoutines';
import { fetchFeatureFlags } from 'services/FeatureFlagsService';

// Types

export const types = {
    FETCH_FEATURE_FLAGS: createFetchingActionTypes('featureflags/FETCH_FEATURE_FLAGS'),
};

// Actions

export const fetchFeatureFlagsThunk = () => {
    return async (dispatch) => {
        dispatch({ type: types.FETCH_FEATURE_FLAGS.REQUEST });

        try {
            const result = await fetchFeatureFlags();
            dispatch({
                type: types.FETCH_FEATURE_FLAGS.SUCCESS,
                response: result.response,
            });
        } catch (error) {
            dispatch({ type: types.FETCH_FEATURE_FLAGS.FAILURE, error });
        }
    };
};

// Reducers

const featureFlags = (state = [], action) => {
    if (action.type === types.FETCH_FEATURE_FLAGS.SUCCESS) {
        return action.response.featureFlags ?? state;
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
