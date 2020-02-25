import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_TELEMETRY_CONFIG: createFetchingActionTypes('telemetry/FETCH_TELEMETRY_CONFIG'),
    SAVE_TELEMETRY_CONFIG: 'telemetry/SAVE_TELEMETRY_CONFIG'
};

// Actions

export const actions = {
    fetchTelemetryConfig: createFetchingActions(types.FETCH_TELEMETRY_CONFIG),
    saveTelemetryConfig: telemetryConfig => ({ type: types.SAVE_TELEMETRY_CONFIG, telemetryConfig })
};

// Reducers

const telemetryConfig = (state = {}, action) => {
    if (action.type === types.FETCH_TELEMETRY_CONFIG.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = telemetryConfig;

// Selectors

const getTelemetryConfig = state => {
    return state;
};

export const selectors = {
    getTelemetryConfig
};

export default reducer;
