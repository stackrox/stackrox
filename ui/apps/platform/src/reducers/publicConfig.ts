import { Reducer, combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { fetchPublicConfig } from 'services/SystemConfigService';
import { PublicConfig } from 'types/config.proto';
import {
    FailureAction,
    FetchingAction,
    createFetchingActionTypes,
} from 'utils/fetchingReduxRoutines';

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

const isLoadingPublicConfig: Reducer<boolean> = (state = true, action) => {
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

const publicConfigInitialState: PublicConfig = {
    footer: null,
    header: null,
    loginNotice: null,
};

const publicConfig: Reducer<PublicConfig, FetchingAction<{ response: PublicConfig }>> = (
    state = publicConfigInitialState,
    action
) => {
    if (action.type === types.FETCH_PUBLIC_CONFIG.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const publicConfigError: Reducer<Error | null, ReturnType<FailureAction>> = (
    state = null,
    action
) => {
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

export const selectors = {
    isLoadingPublicConfigSelector,
    publicConfigErrorSelector,
    publicConfigFooterSelector,
    publicConfigHeaderSelector,
    publicConfigLoginNoticeSelector,
};

export default reducer;
