import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_NOTIFIERS: createFetchingActionTypes('notifiers/FETCH_NOTIFIERS'),
    FETCH_IMAGE_INTEGRATIONS: createFetchingActionTypes(
        'imageIntegrations/FETCH_IMAGE_INTEGRATIONS'
    )
};

// Actions

export const actions = {
    fetchNotifiers: createFetchingActions(types.FETCH_NOTIFIERS),
    fetchImageIntegrations: createFetchingActions(types.FETCH_IMAGE_INTEGRATIONS)
};

// Reducers

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
    notifiers,
    imageIntegrations
});

// Selectors

const getNotifiers = state => state.notifiers;
const getImageIntegrations = state => state.imageIntegrations;

export const selectors = {
    getNotifiers,
    getImageIntegrations
};

export default reducer;
