import { combineReducers } from 'redux';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export const types = {
    FETCH_FEATURE_FLAGS: createFetchingActionTypes('featureflags/FETCH_FEATURE_FLAGS')
};

export const actions = {
    fetchFeatureFlags: createFetchingActions(types.FETCH_FEATURE_FLAGS)
};

const featureFlags = (state = [], action) => {
    if (action.type === types.FETCH_FEATURE_FLAGS.SUCCESS) {
        return action.response.featureFlags;
    }
    return state;
};

const reducer = combineReducers({
    featureFlags
});

const getFeatureFlags = state => state.featureFlags;

export const selectors = {
    getFeatureFlags
};

export default reducer;
