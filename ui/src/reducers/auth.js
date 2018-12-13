import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import { availableAuthProviders } from 'constants/accessControl';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Helper functions

const filterAuthProviders = providers => {
    const availableTypes = availableAuthProviders.map(provider => provider.value);
    const filteredProviders = providers.filter(
        provider => availableTypes.indexOf(provider.type) !== -1
    );
    return filteredProviders;
};

// Action types

export const types = {
    FETCH_AUTH_PROVIDERS: createFetchingActionTypes('auth/FETCH_AUTH_PROVIDERS'),
    SELECTED_AUTH_PROVIDER: 'auth/SELECTED_AUTH_PROVIDER',
    SAVE_AUTH_PROVIDER: 'auth/SAVE_AUTH_PROVIDER',
    DELETE_AUTH_PROVIDER: 'auth/DELETE_AUTH_PROVIDER',
    LOGIN: 'auth/LOGIN',
    LOGOUT: 'auth/LOGOUT',
    GRANT_ANONYMOUS_ACCESS: 'auth/GRANT_ANONYMOUS_ACCESS',
    AUTH_HTTP_ERROR: 'auth/AUTH_HTTP_ERROR',
    AUTH_IDP_ERROR: 'auth/AUTH_IDP_ERROR'
};

// Actions

export const actions = {
    fetchAuthProviders: createFetchingActions(types.FETCH_AUTH_PROVIDERS),
    selectAuthProvider: authProvider => ({
        type: types.SELECTED_AUTH_PROVIDER,
        authProvider
    }),
    saveAuthProvider: authProvider => ({
        type: types.SAVE_AUTH_PROVIDER,
        authProvider
    }),
    deleteAuthProvider: id => ({
        type: types.DELETE_AUTH_PROVIDER,
        id
    }),
    login: () => ({ type: types.LOGIN }),
    logout: () => ({ type: types.LOGOUT }),
    grantAnonymousAccess: () => ({ type: types.GRANT_ANONYMOUS_ACCESS }),
    handleAuthHttpError: error => ({ type: types.AUTH_HTTP_ERROR, error }),
    handleIdpError: error => ({ type: types.AUTH_IDP_ERROR, error })
};

// Reducers

const authProviders = (state = [], action) => {
    if (action.type === types.FETCH_AUTH_PROVIDERS.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const selectedAuthProvider = (state = null, action) => {
    if (action.type === types.FETCH_AUTH_PROVIDERS.SUCCESS && !state) {
        const providers = filterAuthProviders(action.response);
        if (providers.length) {
            return providers[0];
        }
        return state;
    }
    if (action.type === types.SELECTED_AUTH_PROVIDER) {
        return isEqual(action.authProvider, state) ? state : action.authProvider;
    }
    return state;
};

export const AUTH_STATUS = Object.freeze({
    LOADING: 'LOADING',
    LOGGED_IN: 'LOGGED_IN',
    LOGGED_OUT: 'LOGGED_OUT',
    ANONYMOUS_ACCESS: 'ANONYMOUS_ACCESS',
    AUTH_PROVIDERS_LOADING_ERROR: 'AUTH_PROVIDERS_LOADING_ERROR'
});

const authStatus = (state = AUTH_STATUS.LOADING, action) => {
    switch (action.type) {
        case types.LOGIN:
            return AUTH_STATUS.LOGGED_IN;
        case types.LOGOUT:
            return AUTH_STATUS.LOGGED_OUT;
        case types.GRANT_ANONYMOUS_ACCESS:
            return AUTH_STATUS.ANONYMOUS_ACCESS;
        case types.FETCH_AUTH_PROVIDERS.FAILURE:
            return AUTH_STATUS.AUTH_PROVIDERS_LOADING_ERROR;
        default:
            return state;
    }
};

const authProviderResponse = (state = {}, action) => {
    if (action.type === types.AUTH_IDP_ERROR) {
        if (action.error && action.error.error) {
            return action.error;
        }
        return null;
    }
    return state;
};

const reducer = combineReducers({
    authProviders,
    selectedAuthProvider,
    authStatus,
    authProviderResponse
});

export default reducer;

// Selectors

const getAuthProviders = state => state.authProviders;
const getAvailableAuthProviders = state => filterAuthProviders(state.authProviders);
const getSelectedAuthProvider = state => state.selectedAuthProvider;
const getAuthStatus = state => state.authStatus;
const getAuthProviderError = state => state.authProviderResponse;

export const selectors = {
    getAuthProviders,
    getAvailableAuthProviders,
    getSelectedAuthProvider,
    getAuthStatus,
    getAuthProviderError
};
