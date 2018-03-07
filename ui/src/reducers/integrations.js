import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_NOTIFIERS: createFetchingActionTypes('notifiers/FETCH_NOTIFIERS'),
    FETCH_REGISTRIES: createFetchingActionTypes('registries/FETCH_REGISTRIES'),
    FETCH_SCANNERS: createFetchingActionTypes('scanners/FETCH_SCANNERS')
};

// Actions

export const actions = {
    fetchNotifiers: createFetchingActions(types.FETCH_NOTIFIERS),
    fetchRegistries: createFetchingActions(types.FETCH_REGISTRIES),
    fetchScanners: createFetchingActions(types.FETCH_SCANNERS)
};

// Reducers

const notifiers = (state = [], action) => {
    if (action.type === types.FETCH_NOTIFIERS.SUCCESS) {
        return isEqual(action.response.notifiers, state) ? state : action.response.notifiers;
    }
    return state;
};

const registries = (state = [], action) => {
    if (action.type === types.FETCH_REGISTRIES.SUCCESS) {
        return isEqual(action.response.registries, state) ? state : action.response.registries;
    }
    return state;
};

const scanners = (state = [], action) => {
    if (action.type === types.FETCH_SCANNERS.SUCCESS) {
        return isEqual(action.response.scanners, state) ? state : action.response.scanners;
    }
    return state;
};

const reducer = combineReducers({
    notifiers,
    registries,
    scanners
});

// Selectors

const getNotifiers = state => state.notifiers;
const getRegistries = state => state.registries;
const getScanners = state => state.scanners;

export const selectors = {
    getNotifiers,
    getRegistries,
    getScanners
};

export default reducer;
