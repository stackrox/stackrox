import { Reducer, combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { fetchPublicConfig } from 'services/SystemConfigService';
import { PublicConfig } from 'types/config.proto';
import { PrefixedAction } from 'utils/fetchingReduxRoutines';
import { fetchTelemetryConfigThunk } from './telemetryConfig';

// Action types

export type PublicConfigAction = PrefixedAction<'config/FETCH_PUBLIC_CONFIG', PublicConfig>;

// Thunk

export const fetchPublicConfigThunk = () => {
    return async (dispatch) => {
        dispatch({ type: 'config/FETCH_PUBLIC_CONFIG_REQUEST' });

        try {
            const result = await fetchPublicConfig();
            dispatch({
                type: 'config/FETCH_PUBLIC_CONFIG_SUCCESS',
                response: result.response,
            });
        } catch (error) {
            dispatch({ type: 'config/FETCH_PUBLIC_CONFIG_FAILURE', error });
        } finally {
            dispatch(fetchTelemetryConfigThunk());
        }
    };
};

// Reducers

const isLoadingPublicConfig: Reducer<boolean, PublicConfigAction> = (state = true, action) => {
    // Initialize true for edge case before authSagas call fetchUserRolePermissions action.
    switch (action.type) {
        case 'config/FETCH_PUBLIC_CONFIG_REQUEST':
            return true;

        case 'config/FETCH_PUBLIC_CONFIG_FAILURE':
        case 'config/FETCH_PUBLIC_CONFIG_SUCCESS':
            return false;

        default:
            return state;
    }
};

const publicConfigInitialState: PublicConfig = {
    footer: null,
    header: null,
    loginNotice: null,
    telemetry: null,
};

const publicConfig: Reducer<PublicConfig, PublicConfigAction> = (
    state = publicConfigInitialState,
    action
) => {
    switch (action.type) {
        case 'config/FETCH_PUBLIC_CONFIG_SUCCESS':
            return isEqual(action.response, state) ? state : action.response;
        default:
            return state;
    }
};

const publicConfigError: Reducer<Error | null, PublicConfigAction> = (state = null, action) => {
    switch (action.type) {
        case 'config/FETCH_PUBLIC_CONFIG_REQUEST':
        case 'config/FETCH_PUBLIC_CONFIG_SUCCESS':
            return null;

        case 'config/FETCH_PUBLIC_CONFIG_FAILURE':
            return action.error;

        default:
            return state;
    }
};

const reducer = combineReducers({
    isLoadingPublicConfig,
    publicConfig,
    publicConfigError,
});

// Selectors

type State = ReturnType<typeof reducer>;

const isLoadingPublicConfigSelector = (state: State) => state.isLoadingPublicConfig;
const publicConfigErrorSelector = (state: State) => state.publicConfigError;
const publicConfigFooterSelector = (state: State) => state.publicConfig.footer;
const publicConfigHeaderSelector = (state: State) => state.publicConfig.header;
const publicConfigLoginNoticeSelector = (state: State) => state.publicConfig.loginNotice;
const publicConfigTelemetrySelector = (state: State) => state.publicConfig.telemetry;

export const selectors = {
    isLoadingPublicConfigSelector,
    publicConfigErrorSelector,
    publicConfigFooterSelector,
    publicConfigHeaderSelector,
    publicConfigLoginNoticeSelector,
    publicConfigTelemetrySelector,
};

export default reducer;
