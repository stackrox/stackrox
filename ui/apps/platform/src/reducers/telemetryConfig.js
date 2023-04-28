import { combineReducers } from 'redux';

import { createFetchingActionTypes } from 'utils/fetchingReduxRoutines';
import { fetchTelemetryConfig } from 'services/TelemetryConfigService';

import { initializeAnalytics } from 'global/initializeAnalytics';

// Types

export const types = {
    FETCH_TELEMETRY_CONFIG: createFetchingActionTypes('telemetryConfig/FETCH_TELEMETRY_CONFIG'),
};

// Actions

export const fetchTelemetryConfigThunk = () => {
    return async (dispatch, getState) => {
        dispatch({ type: types.FETCH_TELEMETRY_CONFIG.REQUEST });

        try {
            const result = await fetchTelemetryConfig();
            const { app: appState } = getState();
            const telemetryEnabled =
                appState?.publicConfig?.publicConfig?.telemetry?.enabled !== false;
            if (telemetryEnabled) {
                initializeAnalytics(result.response.storageKeyV1, result.response.userId);
            }

            dispatch({
                type: types.FETCH_TELEMETRY_CONFIG.SUCCESS,
                response: result.response,
            });
        } catch (error) {
            dispatch({ type: types.FETCH_TELEMETRY_CONFIG.FAILURE, error });
        }
    };
};

// Reducers

const telemetryConfig = (state = [], action) => {
    if (action.type === types.FETCH_TELEMETRY_CONFIG.SUCCESS) {
        return action.response ?? state;
    }
    return state;
};

const error = (state = null, action) => {
    switch (action.type) {
        case types.FETCH_TELEMETRY_CONFIG.REQUEST:
        case types.FETCH_TELEMETRY_CONFIG.SUCCESS:
            return null;

        case types.FETCH_TELEMETRY_CONFIG.FAILURE:
            return action.error;

        default:
            return state;
    }
};

const isLoading = (state = true, action) => {
    switch (action.type) {
        case types.FETCH_TELEMETRY_CONFIG.REQUEST:
            return true;

        case types.FETCH_TELEMETRY_CONFIG.FAILURE:
        case types.FETCH_TELEMETRY_CONFIG.SUCCESS:
            return false;

        default:
            return state;
    }
};

const isConfigured = (state = false, action) => {
    switch (action.type) {
        case types.FETCH_TELEMETRY_CONFIG.SUCCESS:
            return true;

        case types.FETCH_TELEMETRY_CONFIG.FAILURE:
        case types.FETCH_TELEMETRY_CONFIG.REQUEST:
            return false;

        default:
            return state;
    }
};

const reducer = combineReducers({
    telemetryConfig,
    error,
    isLoading,
    isConfigured,
});

const getTelemetryConfig = (state) => state.telemetryConfig;
const getTelemetryConfigError = (state) => state.error;
const getIsLoadingTelemetryConfig = (state) => state.isLoading;
const getIsTelemetryConfigured = (state) => state.isConfigured;

export const selectors = {
    getTelemetryConfig,
    getTelemetryConfigError,
    getIsLoadingTelemetryConfig,
    getIsTelemetryConfigured,
};

export default reducer;
