import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_DNR_INTEGRATIONS: createFetchingActionTypes('dnrIntegrations/FETCH_DNR_INTEGRATIONS'),
    FETCH_NOTIFIERS: createFetchingActionTypes('notifiers/FETCH_NOTIFIERS'),
    FETCH_IMAGE_INTEGRATIONS: createFetchingActionTypes(
        'imageIntegrations/FETCH_IMAGE_INTEGRATIONS'
    )
};

// Actions

export const actions = {
    fetchDNRIntegrations: createFetchingActions(types.FETCH_DNR_INTEGRATIONS),
    fetchNotifiers: createFetchingActions(types.FETCH_NOTIFIERS),
    fetchImageIntegrations: createFetchingActions(types.FETCH_IMAGE_INTEGRATIONS)
};

// Reducers

const dnrIntegrations = (state = [], action) => {
    if (action.type === types.FETCH_DNR_INTEGRATIONS.SUCCESS) {
        return isEqual(action.response.results, state) ? state : action.response.results;
    }
    return state;
};

const notifiers = (state = [], action) => {
    if (action.type === types.FETCH_NOTIFIERS.SUCCESS) {
        return isEqual(action.response.notifiers, state) ? state : action.response.notifiers;
    }
    return state;
};

const imageIntegrations = (state = [], action) => {
    if (action.type === types.FETCH_IMAGE_INTEGRATIONS.SUCCESS) {
        return isEqual(action.response.integrations, state) ? state : action.response.integrations;
    }
    return state;
};

const reducer = combineReducers({
    dnrIntegrations,
    notifiers,
    imageIntegrations
});

// Selectors

const getDNRIntegrations = state => state.dnrIntegrations;
const getNotifiers = state => state.notifiers;
const getImageIntegrations = state => state.imageIntegrations;

export const selectors = {
    getDNRIntegrations,
    getNotifiers,
    getImageIntegrations
};

export default reducer;
