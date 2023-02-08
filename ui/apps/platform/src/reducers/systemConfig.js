import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes } from 'utils/fetchingReduxRoutines';
import { fetchPublicConfig } from 'services/SystemConfigService';

// Action types

export const types = {
    FETCH_PUBLIC_CONFIG: createFetchingActionTypes('notifiers/FETCH_PUBLIC_CONFIG'),
};

// Actions

export const fetchPublicConfigThunk = () => {
    return async (dispatch) => {
        dispatch({ type: types.FETCH_PUBLIC_CONFIG.REQUEST });

        try {
            const result = await fetchPublicConfig();
            dispatch({
                type: types.FETCH_PUBLIC_CONFIG.SUCCESS,
                response: result.response,
            });
        } catch (error) {
            dispatch({ type: types.FETCH_PUBLIC_CONFIG.FAILURE, error });
        }
    };
};

// Reducers

const publicConfig = (state = { footer: null, header: null, loginNotice: null }, action) => {
    if (action.type === types.FETCH_PUBLIC_CONFIG.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const error = (state = null, action) => {
    switch (action.type) {
        case types.FETCH_PUBLIC_CONFIG.REQUEST:
        case types.FETCH_PUBLIC_CONFIG.SUCCESS:
            return null;

        case types.FETCH_PUBLIC_CONFIG.FAILURE:
            return action.error;

        default:
            return state;
    }
};

const isLoading = (state = true, action) => {
    // Initialize true for edge case before authSagas call fetchUserRolePermissions action.
    switch (action.type) {
        case types.FETCH_PUBLIC_CONFIG.REQUEST:
            return true;

        case types.FETCH_PUBLIC_CONFIG.FAILURE:
        case types.FETCH_PUBLIC_CONFIG.SUCCESS:
            return false;

        default:
            return state;
    }
};

const reducer = combineReducers({
    publicConfig,
    error,
    isLoading,
});

// Selectors

const getPublicConfig = (state) => state.publicConfig;
const getPublicConfigFooter = (state) => state.publicConfig.footer;
const getPublicConfigHeader = (state) => state.publicConfig.header;
const getPublicConfigLoginNotice = (state) => state.publicConfig.loginNotice;
const getPublicConfigError = (state) => state.error;
const getIsLoadingPublicConfig = (state) => state.isLoading;

export const selectors = {
    getPublicConfig,
    getPublicConfigFooter,
    getPublicConfigHeader,
    getPublicConfigLoginNotice,
    getPublicConfigError,
    getIsLoadingPublicConfig,
};

export default reducer;
