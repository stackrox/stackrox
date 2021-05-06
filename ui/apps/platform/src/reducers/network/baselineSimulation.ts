import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';
import { BaselineComparisonsResponse } from 'Containers/Network/networkTypes';

export type BaselineSimulationOptions = {
    excludePortsAndProtocols: boolean;
};

export type StartBaselineSimulationAction = {
    type: 'network/START_BASELINE_SIMULATION';
    options: BaselineSimulationOptions;
};

export type StopBaselineSimulationAction = {
    type: 'network/STOP_BASELINE_SIMULATION';
};

export type BaselineSimulationState = {
    isOn: boolean;
    options: { excludePortsAndProtocols: boolean };
    baselineComparisons: BaselineComparisonsResult;
};

export type BaselineComparisonsAction = {
    type:
        | 'network/FETCH_BASELINE_COMPARISONS_REQUEST'
        | 'network/FETCH_BASELINE_COMPARISONS_SUCCESS'
        | 'network/FETCH_BASELINE_COMPARISONS_FAILURE';
    response: BaselineComparisonsResponse;
    error: Error | null;
};

export type BaselineComparisonsResult = {
    isLoading: boolean;
    data: BaselineComparisonsResponse;
    error: Error | null;
};

const baselineComparisonsDefaultState = {
    isLoading: false,
    data: { added: [], removed: [], reconciled: [] },
    error: null,
};

// Action types
//-------------

export const types = {
    START_BASELINE_SIMULATION: 'network/START_BASELINE_SIMULATION',
    STOP_BASELINE_SIMULATION: 'network/STOP_BASELINE_SIMULATION',
    FETCH_BASELINE_COMPARISONS: createFetchingActionTypes('network/FETCH_BASELINE_COMPARISONS'),
};

// Actions
//---------

export const actions = {
    startBaselineSimulation: (
        options: BaselineSimulationOptions
    ): StartBaselineSimulationAction => ({
        type: 'network/START_BASELINE_SIMULATION',
        options,
    }),
    stopBaselineSimulation: (): StopBaselineSimulationAction => ({
        type: 'network/STOP_BASELINE_SIMULATION',
    }),
    fetchBaselineComparisons: createFetchingActions(types.FETCH_BASELINE_COMPARISONS),
};

// Reducers
// If adding a reducer, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const isOn = (
    state = false,
    action: StartBaselineSimulationAction | StopBaselineSimulationAction
) => {
    if (action.type === 'network/START_BASELINE_SIMULATION') {
        const newState = true;
        return isEqual(newState, state) ? state : newState;
    }
    if (action.type === 'network/STOP_BASELINE_SIMULATION') {
        const newState = false;
        return isEqual(newState, state) ? state : newState;
    }
    return state;
};

const options = (
    state: BaselineSimulationOptions = { excludePortsAndProtocols: false },
    action: StartBaselineSimulationAction | StopBaselineSimulationAction
) => {
    if (action.type === 'network/START_BASELINE_SIMULATION') {
        const newState = action.options;
        return isEqual(newState, state) ? state : newState;
    }
    if (action.type === 'network/STOP_BASELINE_SIMULATION') {
        const newState = { excludePortsAndProtocols: false };
        return isEqual(newState, state) ? state : newState;
    }
    return state;
};

const baselineComparisons = (
    state: BaselineComparisonsResult = baselineComparisonsDefaultState,
    action: BaselineComparisonsAction
): BaselineComparisonsResult => {
    const { type } = action;
    if (type === 'network/FETCH_BASELINE_COMPARISONS_REQUEST') {
        const newState = { ...baselineComparisonsDefaultState, isLoading: true };
        return isEqual(newState, state) ? state : newState;
    }
    if (type === 'network/FETCH_BASELINE_COMPARISONS_SUCCESS') {
        const { response } = action;
        const newState = { ...baselineComparisonsDefaultState, data: response };
        return isEqual(newState, state) ? state : newState;
    }
    if (type === 'network/FETCH_BASELINE_COMPARISONS_FAILURE') {
        const { error } = action;
        const newState = { ...baselineComparisonsDefaultState, error };
        return isEqual(newState, state) ? state : newState;
    }
    return state;
};

const reducer = combineReducers({
    isOn,
    options,
    baselineComparisons,
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const getIsBaselineSimulationOn = (state: BaselineSimulationState): boolean => state.isOn;
const getBaselineSimulationOptions = (state: BaselineSimulationState): BaselineSimulationOptions =>
    state.options;
const getBaselineComparisons = (state: BaselineSimulationState): BaselineComparisonsResult =>
    state.baselineComparisons;

export const selectors = {
    getIsBaselineSimulationOn,
    getBaselineSimulationOptions,
    getBaselineComparisons,
};
