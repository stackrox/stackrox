import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_PUBLIC_CONFIG: createFetchingActionTypes('notifiers/FETCH_PUBLIC_CONFIG'),
};

// Actions

export const actions = {
    fetchPublicConfig: createFetchingActions(types.FETCH_PUBLIC_CONFIG),
};

// Reducers

const publicConfig = (state = {}, action) => {
    if (action.type === types.FETCH_PUBLIC_CONFIG.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = combineReducers({
    publicConfig,
});

// Selectors

const getPublicConfig = (state) => state.publicConfig;

export const selectors = {
    getPublicConfig,
};

export default reducer;
