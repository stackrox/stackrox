import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';
import { createFetchingActions, createFetchingActionTypes } from 'utils/fetchingReduxRoutines';

// Helper functions

export const filterAuthProviders = (providers, availableTypes) => {
    const availableTypeNames = availableTypes?.map((provider) => provider.value);
    return providers.filter((provider) => availableTypeNames?.indexOf(provider.type) !== -1);
};

// Action types

export const types = {
    FETCH_AUTH_PROVIDERS: createFetchingActionTypes('auth/FETCH_AUTH_PROVIDERS'),
    FETCH_LOGIN_AUTH_PROVIDERS: createFetchingActionTypes('auth/FETCH_LOGIN_AUTH_PROVIDERS'),
    FETCH_AVAILABLE_PROVIDER_TYPES: 'auth/FETCH_AVAILABLE_PROVIDER_TYPES',
    SELECTED_AUTH_PROVIDER: 'auth/SELECTED_AUTH_PROVIDER',
    SAVE_AUTH_PROVIDER: 'auth/SAVE_AUTH_PROVIDER',
    DELETE_AUTH_PROVIDER: 'auth/DELETE_AUTH_PROVIDER',
    LOGIN: 'auth/LOGIN',
    LOGOUT: 'auth/LOGOUT',
    GRANT_ANONYMOUS_ACCESS: 'auth/GRANT_ANONYMOUS_ACCESS',
    AUTH_HTTP_ERROR: 'auth/AUTH_HTTP_ERROR',
    AUTH_IDP_ERROR: 'auth/AUTH_IDP_ERROR',
    SET_AUTH_PROVIDER_EDITING_STATE: 'auth/SET_AUTH_PROVIDER_EDITING_STATE',
    SET_SAVE_AUTH_PROVIDER_STATUS: 'auth/SET_AUTH_PROVIDER_ERROR',
    SET_AUTH_PROVIDER_TEST_RESULTS: 'auth/SET_AUTH_PROVIDER_TEST_RESULTS',
};

// Actions

export const actions = {
    fetchAuthProviders: createFetchingActions(types.FETCH_AUTH_PROVIDERS),
    fetchLoginAuthProviders: createFetchingActions(types.FETCH_LOGIN_AUTH_PROVIDERS),
    selectAuthProvider: (authProvider) => ({
        type: types.SELECTED_AUTH_PROVIDER,
        authProvider,
    }),
    saveAuthProvider: (authProvider) => ({
        type: types.SAVE_AUTH_PROVIDER,
        authProvider,
    }),
    deleteAuthProvider: (id) => ({
        type: types.DELETE_AUTH_PROVIDER,
        id,
    }),
    setAuthProviderEditingState: (value) => ({
        type: types.SET_AUTH_PROVIDER_EDITING_STATE,
        value,
    }),
    setSaveAuthProviderStatus: (status) => ({
        type: types.SET_SAVE_AUTH_PROVIDER_STATUS,
        status,
    }),
    setAuthProviderTestResults: (value) => ({
        type: types.SET_AUTH_PROVIDER_TEST_RESULTS,
        value,
    }),
    setAvailableProviderTypes: (value) => ({
        type: types.FETCH_AVAILABLE_PROVIDER_TYPES,
        value,
    }),
    login: (userData) => ({ type: types.LOGIN, userData }),
    logout: () => ({ type: types.LOGOUT }),
    grantAnonymousAccess: () => ({ type: types.GRANT_ANONYMOUS_ACCESS }),
    handleAuthHttpError: (error) => ({ type: types.AUTH_HTTP_ERROR, error }),
    handleIdpError: (error) => ({ type: types.AUTH_IDP_ERROR, error }),
};

// Reducers

const currentUser = (state = {}, action) => {
    if (action.type === types.LOGIN) {
        return isEqual(action.userData, state) ? state : action.userData;
    }
    return state;
};

const authProviders = (state = [], action) => {
    if (action.type === types.FETCH_AUTH_PROVIDERS.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const authProviderTestResults = (state = {}, action) => {
    if (action.type === types.SET_AUTH_PROVIDER_TEST_RESULTS) {
        return isEqual(action.value, state) ? state : action.value;
    }
    return state;
};

const loginAuthProviders = (state = [], action) => {
    if (action.type === types.FETCH_LOGIN_AUTH_PROVIDERS.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const selectedAuthProvider = (state = null, action) => {
    if (action.type === types.FETCH_AUTH_PROVIDERS.SUCCESS) {
        const providers = filterAuthProviders(action.response, state?.supportedAuthProviders);
        if (state?.id && !providers.find((provider) => provider.id === state.id)) {
            // the selected auth provider isn't anymore in the list of auth providers => deselect
            return null;
        }
    }
    if (action.type === types.SELECTED_AUTH_PROVIDER) {
        if (!action.authProvider) {
            return null;
        }
        return isEqual(action.authProvider, state) ? state : action.authProvider;
    }
    if (action.type === types.DELETE_AUTH_PROVIDER && state?.id === action.id) {
        return null; // selected auth provider got deleted => deselect
    }
    return state;
};

const isEditingAuthProvider = (state = false, action) => {
    if (action.type === types.SET_AUTH_PROVIDER_EDITING_STATE) {
        return isEqual(action.value, state) ? state : action.value;
    }
    return state;
};

export const AUTH_STATUS = Object.freeze({
    LOADING: 'LOADING',
    LOGGED_IN: 'LOGGED_IN',
    LOGGED_OUT: 'LOGGED_OUT',
    ANONYMOUS_ACCESS: 'ANONYMOUS_ACCESS',
    AUTH_PROVIDERS_LOADING_ERROR: 'AUTH_PROVIDERS_LOADING_ERROR',
    LOGIN_AUTH_PROVIDERS_LOADING_ERROR: 'LOGIN_AUTH_PROVIDERS_LOADING_ERROR',
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
        case types.FETCH_LOGIN_AUTH_PROVIDERS.FAILURE:
            return AUTH_STATUS.LOGIN_AUTH_PROVIDERS_LOADING_ERROR;
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

const saveAuthProviderStatus = (state = null, action) => {
    if (action.type === types.SET_SAVE_AUTH_PROVIDER_STATUS) {
        if (action.status && (action.status.message || action.status.status)) {
            return action.status;
        }
        return null;
    }
    return state;
};

const availableProviderTypes = (state = [], action) => {
    if (action.type === types.FETCH_AVAILABLE_PROVIDER_TYPES) {
        return isEqual(action.value, state) ? state : action.value;
    }
    return state;
};

const reducer = combineReducers({
    authProviders,
    authProviderTestResults,
    loginAuthProviders,
    selectedAuthProvider,
    authStatus,
    authProviderResponse,
    isEditingAuthProvider,
    saveAuthProviderStatus,
    currentUser,
    availableProviderTypes,
});

export default reducer;

// Selectors

const getAuthProviders = (state) => state.authProviders;
const getLoginAuthProviders = (state) => state.loginAuthProviders;
const getLoginAuthProviderTestResults = (state) => state.authProviderTestResults;
const getAvailableAuthProviders = (state) =>
    filterAuthProviders(state.authProviders, state.availableProviderTypes);
const getSelectedAuthProvider = (state) => state.selectedAuthProvider;
const getAuthStatus = (state) => state.authStatus;
const getAuthProviderError = (state) => state.authProviderResponse;
const getAuthProviderEditingState = (state) => state.isEditingAuthProvider;
const getSaveAuthProviderStatus = (state) => state.saveAuthProviderStatus;
const getCurrentUser = (state) => state.currentUser;
const getAvailableProviderTypes = (state) => state.availableProviderTypes;

export const selectors = {
    getAuthProviders,
    getLoginAuthProviders,
    getLoginAuthProviderTestResults,
    getAvailableAuthProviders,
    getSelectedAuthProvider,
    getAuthStatus,
    getAuthProviderError,
    getAuthProviderEditingState,
    getSaveAuthProviderStatus,
    getCurrentUser,
    getAvailableProviderTypes,
};
