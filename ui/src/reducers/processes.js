import { combineReducers } from 'redux';
import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_PROCESSES: createFetchingActionTypes('processes/FETCH_PROCESSES')
};

// Actions

export const actions = {
    fetchProcesses: createFetchingActions(types.FETCH_PROCESSES)
};

// Reducers

const byDeployment = (state = {}, action) => {
    if (action.type === types.FETCH_PROCESSES.SUCCESS) {
        return action.response;
    }
    return state;
};

const reducer = combineReducers({
    byDeployment
});

export default reducer;

// Selectors

const getProcessesByDeployment = state => state.byDeployment;

export const selectors = {
    getProcessesByDeployment
};
