import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes } from 'utils/fetchingReduxRoutines';
import { fetchPublicConfig as fetchPublicConfigService } from 'services/SystemConfigService';

// Action types

export const types = {
    FETCH_PUBLIC_CONFIG: createFetchingActionTypes('notifiers/FETCH_PUBLIC_CONFIG'),
};

// Actions

export const fetchPublicConfig = () => {
    return async (dispatch) => {
        dispatch({ type: types.FETCH_PUBLIC_CONFIG.REQUEST });

        try {
            const result = await fetchPublicConfigService();
            dispatch({
                type: types.FETCH_PUBLIC_CONFIG.SUCCESS,
                response: result.response,
            });
        } catch (e) {
            dispatch({ type: types.FETCH_PUBLIC_CONFIG.FAILURE, payload: e });
        }
    };
};

// Reducers

const publicConfig = (state = {}, action) => {
    if (action.type === types.FETCH_PUBLIC_CONFIG.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = combineReducers({
    publicConfig,
});

// Selectors

const getPublicConfig = (state) => state.publicConfig;

export const selectors = {
    getPublicConfig,
};

export default reducer;
