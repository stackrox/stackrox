import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_AUTH_PROVIDERS: createFetchingActionTypes('auth/FETCH_AUTH_PROVIDERS')
};

// Actions

export const actions = {
    fetchAuthProviders: createFetchingActions(types.FETCH_AUTH_PROVIDERS)
};

// Reducers

const authProviders = (state = [], action) => {
    if (action.type === types.FETCH_AUTH_PROVIDERS.SUCCESS) {
        return isEqual(action.response.authProviders, state)
            ? state
            : action.response.authProviders;
    }
    return state;
};

const reducer = combineReducers({
    authProviders
});

export default reducer;

// Selectors

const getAuthProviders = state => state.authProviders;

export const selectors = {
    getAuthProviders
};
