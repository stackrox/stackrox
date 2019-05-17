import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_SYSTEM_CONFIG: createFetchingActionTypes('notifiers/FETCH_SYSTEM_CONFIG'),
    FETCH_PUBLIC_CONFIG: createFetchingActionTypes('notifiers/FETCH_PUBLIC_CONFIG'),
    SAVE_SYSTEM_CONFIG: 'integrations/SAVE_SYSTEM_CONFIG'
};

// Actions

export const actions = {
    fetchSystemConfig: createFetchingActions(types.FETCH_SYSTEM_CONFIG),
    fetchPublicConfig: createFetchingActions(types.FETCH_PUBLIC_CONFIG),
    saveSystemConfig: systemConfig => ({ type: types.SAVE_SYSTEM_CONFIG, systemConfig })
};

// Reducers

const systemConfig = (state = [], action) => {
    if (action.type === types.FETCH_SYSTEM_CONFIG.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const publicConfig = (state = {}, action) => {
    if (action.type === types.FETCH_PUBLIC_CONFIG.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = combineReducers({
    systemConfig,
    publicConfig
});

// Selectors

const getSystemConfig = state => state.systemConfig;
const getPublicConfig = state => state.publicConfig;

export const selectors = {
    getSystemConfig,
    getPublicConfig
};

export default reducer;
