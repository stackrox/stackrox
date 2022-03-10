import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_AUTH_PLUGINS: createFetchingActionTypes('authPlugins/FETCH_AUTH_PLUGINS'),
    FETCH_NOTIFIERS: createFetchingActionTypes('notifiers/FETCH_NOTIFIERS'),
    FETCH_BACKUPS: createFetchingActionTypes('backups/FETCH_BACKUPS'),
    TRIGGER_BACKUP: 'integrations/TRIGGER_BACKUP',
    FETCH_IMAGE_INTEGRATIONS: createFetchingActionTypes(
        'imageIntegrations/FETCH_IMAGE_INTEGRATIONS'
    ),
    FETCH_SIGNATURE_INTEGRATIONS: createFetchingActionTypes(
        'signatureIntegrations/FETCH_SIGNATURE_INTEGRATIONS'
    ),
    TEST_INTEGRATION: 'integrations/TEST_INTEGRATION',
    DELETE_INTEGRATIONS: 'integrations/DELETE_INTEGRATIONS',
    SAVE_INTEGRATION: createFetchingActionTypes('integrations/SAVE_INTEGRATION'),
    SET_CREATE_STATE: 'integrations/SET_CREATE_STATE',
};

// Actions

export const actions = {
    fetchAuthPlugins: createFetchingActions(types.FETCH_AUTH_PLUGINS),
    fetchNotifiers: createFetchingActions(types.FETCH_NOTIFIERS),
    fetchBackups: createFetchingActions(types.FETCH_BACKUPS),
    fetchImageIntegrations: createFetchingActions(types.FETCH_IMAGE_INTEGRATIONS),
    fetchSignatureIntegrations: createFetchingActions(types.FETCH_SIGNATURE_INTEGRATIONS),
    testIntegration: (source, integration, options) => ({
        type: types.TEST_INTEGRATION,
        source,
        integration,
        options,
    }),
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
    saveIntegration: createFetchingActions(types.SAVE_INTEGRATION),
    setCreateState: (state) => ({
        type: types.SET_CREATE_STATE,
        state,
    }),
};

// Reducers

const authPlugins = (state = [], action) => {
    if (action.type === types.FETCH_AUTH_PLUGINS.SUCCESS) {
        return isEqual(action.response.configs, state) ? state : action.response.configs;
    }
    return state;
};

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

const isCreating = (state = false, action) => {
    if (action.type === types.SET_CREATE_STATE) {
        return action.state;
    }
    return state;
};

const reducer = combineReducers({
    authPlugins,
    backups,
    notifiers,
    imageIntegrations,
    signatureIntegrations,
    isCreating,
});

// Selectors

const getAuthPlugins = (state) => state.authPlugins;
const getBackups = (state) => state.backups;
const getNotifiers = (state) => state.notifiers;
const getImageIntegrations = (state) => state.imageIntegrations;
const getCreationState = (state) => state.isCreating;
const getSignatureIntegrations = (state) => state.signatureIntegrations;

export const selectors = {
    getAuthPlugins,
    getBackups,
    getNotifiers,
    getImageIntegrations,
    getCreationState,
    getSignatureIntegrations,
};

export default reducer;
