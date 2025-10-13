import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_NOTIFIERS: createFetchingActionTypes('notifiers/FETCH_NOTIFIERS'),
    FETCH_BACKUPS: createFetchingActionTypes('backups/FETCH_BACKUPS'),
    TRIGGER_BACKUP: 'integrations/TRIGGER_BACKUP',
    FETCH_IMAGE_INTEGRATIONS: createFetchingActionTypes(
        'imageIntegrations/FETCH_IMAGE_INTEGRATIONS'
    ),
    FETCH_SIGNATURE_INTEGRATIONS: createFetchingActionTypes(
        'signatureIntegrations/FETCH_SIGNATURE_INTEGRATIONS'
    ),
    DELETE_INTEGRATIONS: 'integrations/DELETE_INTEGRATIONS',
};

// Actions

export const actions = {
    fetchNotifiers: createFetchingActions(types.FETCH_NOTIFIERS),
    fetchBackups: createFetchingActions(types.FETCH_BACKUPS),
    fetchImageIntegrations: createFetchingActions(types.FETCH_IMAGE_INTEGRATIONS),
    fetchSignatureIntegrations: createFetchingActions(types.FETCH_SIGNATURE_INTEGRATIONS),
    deleteIntegrations: (source, sourceType, ids) => ({
        type: types.DELETE_INTEGRATIONS,
        source,
        sourceType,
        ids,
    }),
    triggerBackup: (id) => ({
        type: types.TRIGGER_BACKUP,
        id,
    }),
};

// Reducers

const backups = (state = [], action) => {
    if (action.type === types.FETCH_BACKUPS.SUCCESS) {
        return isEqual(action.response.externalBackups, state)
            ? state
            : action.response.externalBackups;
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

const signatureIntegrations = (state = [], action) => {
    if (action.type === types.FETCH_SIGNATURE_INTEGRATIONS.SUCCESS) {
        return isEqual(action.response.integrations, state) ? state : action.response.integrations;
    }
    return state;
};

const reducer = combineReducers({
    backups,
    notifiers,
    imageIntegrations,
    signatureIntegrations,
});

// Selectors

const getBackups = (state) => state.backups;
const getNotifiers = (state) => state.notifiers;
const getImageIntegrations = (state) => state.imageIntegrations;
const getSignatureIntegrations = (state) => state.signatureIntegrations;

export const selectors = {
    getBackups,
    getNotifiers,
    getImageIntegrations,
    getSignatureIntegrations,
};

export default reducer;
