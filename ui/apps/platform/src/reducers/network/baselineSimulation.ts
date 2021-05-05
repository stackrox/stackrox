import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

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
};

// Action types
//-------------

export const types = {
    START_BASELINE_SIMULATION: 'network/START_BASELINE_SIMULATION',
    STOP_BASELINE_SIMULATION: 'network/STOP_BASELINE_SIMULATION',
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

const reducer = combineReducers({
    isOn,
    options,
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const getIsBaselineSimulationOn = (state: BaselineSimulationState): boolean => state.isOn;
const getBaselineSimulationOptions = (state: BaselineSimulationState): BaselineSimulationOptions =>
    state.options;

export const selectors = {
    getIsBaselineSimulationOn,
    getBaselineSimulationOptions,
};
