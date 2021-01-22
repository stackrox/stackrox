import { combineReducers } from 'redux';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_PROCESSES: createFetchingActionTypes('processes/FETCH_PROCESSES'),
    FETCH_PROCESSES_BASELINE: createFetchingActionTypes('processes/FETCH_PROCESSES_BASELINE'),
    ADD_DELETE_PROCESSES: 'processes/ADD_DELETE_PROCESS',
    LOCK_UNLOCK_PROCESSES: 'processes/LOCK_UNLOCK_PROCESS',
};

// Actions

export const actions = {
    fetchProcesses: createFetchingActions(types.FETCH_PROCESSES),
    fetchProcessesBaseline: createFetchingActions(types.FETCH_PROCESSES_BASELINE),
    addDeleteProcesses: (processes) => ({ type: types.ADD_DELETE_PROCESSES, processes }),
    lockUnlockProcesses: (processes) => ({ type: types.LOCK_UNLOCK_PROCESSES, processes }),
};

// Reducers

const byDeployment = (state = {}, action) => {
    if (action.type === types.FETCH_PROCESSES.SUCCESS) {
        return action.response;
    }
    return state;
};

const processesBaselineByDeployment = (state = [], action) => {
    if (action.type === types.FETCH_PROCESSES_BASELINE.SUCCESS) {
        return action.response;
    }
    return state;
};

const reducer = combineReducers({
    byDeployment,
    processesBaselineByDeployment,
});

export default reducer;

// Selectors

const getProcessesByDeployment = (state) => state.byDeployment;
const getProcessesBaselineByDeployment = (state) => state.processesBaselineByDeployment;

export const selectors = {
    getProcessesByDeployment,
    getProcessesBaselineByDeployment,
};
