import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

// Action types

export const types = {
    FETCH_BENCHMARKS: createFetchingActionTypes('benchmarks/FETCH_BENCHMARKS')
};

// Actions

export const actions = {
    fetchBenchmarks: createFetchingActions(types.FETCH_BENCHMARKS)
};

// Reducers

const benchmarks = (state = {}, action) => {
    if (action.type === types.FETCH_BENCHMARKS.SUCCESS) {
        const { response } = action;
        return isEqual(response, state) ? state : response;
    }
    return state;
};

const reducer = combineReducers({
    benchmarks
});

export default reducer;

// Selectors

const getBenchmarks = state => state.benchmarks;

export const selectors = {
    getBenchmarks
};
