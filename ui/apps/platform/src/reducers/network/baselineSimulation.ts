import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { L4Protocol as Protocol } from 'types/networkFlow.proto';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

export type Properties = {
    port: string;
    protocol: Protocol;
    ingress: boolean;
};

export type DeploymentEntity = {
    id: string;
    type: 'DEPLOYMENT';
    deployment: {
        name: string;
        namespace: string;
    };
};

export type ExternalSourceEntity = {
    id: string;
    type: 'EXTERNAL_SOURCE';
    externalSource: {
        name: string;
        cidr: string;
    };
};

export type InternetEntity = {
    id: string;
    type: 'INTERNET';
};

export type AddedRemovedBaselineResponse = {
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity;
    properties: [Properties];
};

export type ReconciledBaselineResponse = {
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity;
    added: [Properties];
    removed: [Properties];
    unchanged: [Properties];
};

export type BaselineComparisonsResponse = {
    added: AddedRemovedBaselineResponse[];
    removed: AddedRemovedBaselineResponse[];
    reconciled: ReconciledBaselineResponse[];
};
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
    baselineComparisons: ComparisonsResult;
    undoComparisons: ComparisonsResult;
    isUndoOn: boolean;
};

export type FetchBaselineActions =
    | 'network/FETCH_BASELINE_COMPARISONS_REQUEST'
    | 'network/FETCH_BASELINE_COMPARISONS_SUCCESS'
    | 'network/FETCH_BASELINE_COMPARISONS_FAILURE';

export type FetchUndoActions =
    | 'network/FETCH_UNDO_COMPARISONS_REQUEST'
    | 'network/FETCH_UNDO_COMPARISONS_SUCCESS'
    | 'network/FETCH_UNDO_COMPARISONS_FAILURE';

export type ComparisonsAction = {
    type: FetchBaselineActions | FetchUndoActions;
    response: BaselineComparisonsResponse;
    error: Error | null;
};

export type ToggleUndoPreviewAction = {
    type: 'network/TOGGLE_UNDO_PREVIEW';
    isOn: boolean;
};

export type ComparisonsResult = {
    isLoading: boolean;
    data: BaselineComparisonsResponse;
    error: Error | null;
};

const comparisonsDefaultState = {
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
    FETCH_UNDO_COMPARISONS: createFetchingActionTypes('network/FETCH_UNDO_COMPARISONS'),
    TOGGLE_UNDO_PREVIEW: 'network/TOGGLE_UNDO_PREVIEW',
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
    fetchUndoComparisons: createFetchingActions(types.FETCH_UNDO_COMPARISONS),
    toggleUndoPreview: (isOn: boolean): ToggleUndoPreviewAction => ({
        type: 'network/TOGGLE_UNDO_PREVIEW',
        isOn,
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

const baselineComparisons = (
    state: ComparisonsResult = comparisonsDefaultState,
    action: ComparisonsAction
): ComparisonsResult => {
    const { type } = action;
    if (type === 'network/FETCH_BASELINE_COMPARISONS_REQUEST') {
        const newState = { ...comparisonsDefaultState, isLoading: true };
        return isEqual(newState, state) ? state : newState;
    }
    if (type === 'network/FETCH_BASELINE_COMPARISONS_SUCCESS') {
        const { response } = action;
        const newState = { ...comparisonsDefaultState, data: response };
        return isEqual(newState, state) ? state : newState;
    }
    if (type === 'network/FETCH_BASELINE_COMPARISONS_FAILURE') {
        const { error } = action;
        const newState = { ...comparisonsDefaultState, error };
        return isEqual(newState, state) ? state : newState;
    }
    return state;
};

const undoComparisons = (
    state: ComparisonsResult = comparisonsDefaultState,
    action: ComparisonsAction
): ComparisonsResult => {
    const { type } = action;
    if (type === 'network/FETCH_UNDO_COMPARISONS_REQUEST') {
        const newState = { ...comparisonsDefaultState, isLoading: true };
        return isEqual(newState, state) ? state : newState;
    }
    if (type === 'network/FETCH_UNDO_COMPARISONS_SUCCESS') {
        const { response } = action;
        const newState = { ...comparisonsDefaultState, data: response };
        return isEqual(newState, state) ? state : newState;
    }
    if (type === 'network/FETCH_UNDO_COMPARISONS_FAILURE') {
        const { error } = action;
        const newState = { ...comparisonsDefaultState, error };
        return isEqual(newState, state) ? state : newState;
    }
    return state;
};

const isUndoOn = (state = false, action: ToggleUndoPreviewAction) => {
    if (action.type === 'network/TOGGLE_UNDO_PREVIEW') {
        const newState = action.isOn;
        return isEqual(newState, state) ? state : newState;
    }
    return state;
};

const reducer = combineReducers({
    isOn,
    options,
    baselineComparisons,
    undoComparisons,
    isUndoOn,
});

export default reducer;

// Selectors
// If adding a selector, you'll need to wire it through reducers/network/reducer.js
//---------------------------------------------------------------------------------

const getIsBaselineSimulationOn = (state: BaselineSimulationState): boolean => state.isOn;
const getBaselineSimulationOptions = (state: BaselineSimulationState): BaselineSimulationOptions =>
    state.options;
const getBaselineComparisons = (state: BaselineSimulationState): ComparisonsResult =>
    state.baselineComparisons;
const getUndoComparisons = (state: BaselineSimulationState): ComparisonsResult =>
    state.undoComparisons;
const getIsUndoOn = (state: BaselineSimulationState): boolean => state.isUndoOn;

export const selectors = {
    getIsBaselineSimulationOn,
    getBaselineSimulationOptions,
    getBaselineComparisons,
    getUndoComparisons,
    getIsUndoOn,
};
