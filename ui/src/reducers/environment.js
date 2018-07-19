import { combineReducers } from 'redux';
import isEqual from 'lodash/isEqual';

import { createFetchingActionTypes, createFetchingActions } from 'utils/fetchingReduxRoutines';

import {
    types as searchTypes,
    getActions as getSearchActions,
    reducers as searchReducers,
    getSelectors as getSearchSelectors
} from 'reducers/pageSearch';

// Action types

export const types = {
    FETCH_NETWORK_GRAPH: createFetchingActionTypes('environment/FETCH_NETWORK_GRAPH'),
    ...searchTypes('environment')
};

// Actions

export const actions = {
    fetchNetworkGraph: createFetchingActions(types.FETCH_NETWORK_GRAPH),
    ...getSearchActions('environment')
};

// Reducers

const networkGraph = (state = {}, action) => {
    if (action.type === types.FETCH_NETWORK_GRAPH.SUCCESS) {
        return isEqual(action.response, state) ? state : action.response;
    }
    return state;
};

const reducer = combineReducers({
    networkGraph,
    ...searchReducers('environment')
});

// Selectors

const getNetworkGraph = state => state.networkGraph;

export const selectors = {
    getNetworkGraph,
    ...getSearchSelectors('environment')
};

export default reducer;
